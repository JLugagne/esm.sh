package server

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/esm-dev/esm.sh/server/storage"
	"github.com/ije/gox/utils"
)

type GitRef struct {
	Ref string
	Sha string
}

// list repo refs using `git ls-remote repo`
func listRepoRefs(repo string) (refs []GitRef, err error) {
	cacheKey := fmt.Sprintf("gh:%s", repo)
	mutex, _ := fetchLock.LoadOrStore(cacheKey, &sync.Mutex{})
	mutex.(*sync.Mutex).Lock()
	defer func() {
		mutex.(*sync.Mutex).Unlock()
	}()

	// check cache firstly
	if cache != nil {
		var data []byte
		data, err = cache.Get(cacheKey)
		if err == nil && json.Unmarshal(data, &refs) == nil {
			return
		}
		if err != nil && err != storage.ErrNotFound && err != storage.ErrExpired {
			log.Error("cache:", err)
		}
	}

	cmd := exec.Command("git", "ls-remote", repo)
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	refs = []GitRef{}
	cmd.Start()
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		sha, ref := utils.SplitByLastByte(line, '\t')
		refs = append(refs, GitRef{
			Ref: ref,
			Sha: sha,
		})
	}
	reader.Close()
	cmd.Wait()

	if cache != nil {
		cache.Set(cacheKey, utils.MustEncodeJSON(refs), 10*time.Minute)
	}
	return
}

func ghInstall(wd, name, hash string) (err error) {
	url := fmt.Sprintf(`https://codeload.github.com/%s/tar.gz/%s`, name, hash)
	res, err := fetch(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	// unzip tarball
	unziped, err := gzip.NewReader(res.Body)
	if err != nil {
		return
	}
	defer unziped.Close()

	// extract tarball
	tr := tar.NewReader(unziped)
	rootDir := path.Join(wd, "node_modules", name)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// strip tarball root dir

		hname := strings.Join(strings.Split(h.Name, "/")[1:], "/")
		if strings.HasPrefix(hname, ".") {
			continue
		}
		fp := path.Join(rootDir, hname)
		if h.Typeflag == tar.TypeDir {
			ensureDir(fp)
			continue
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		f, err := os.OpenFile(fp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, tr)
		f.Close()
		if err != nil {
			return err
		}
	}
	return
}
