package server

import (
	"testing"
)

func TestAliasPrefix(t *testing.T) {
	prefix := encodeAliasPrefix(map[string]string{"a": "b"}, PkgSlice{
		Pkg{Name: "b", Version: "1.0.0"},
		Pkg{Name: "d", Version: "1.0.0"},
		Pkg{Name: "c", Version: "1.0.0"},
	})
	a, d, e := decodeAliasPrefix(prefix)
	if e != nil {
		t.Fatal(e)
	}
	if len(a) != 1 || a["a"] != "b" {
		t.Fatal("invalid alias")
	}
	if len(d) != 3 {
		t.Fatal("invalid deps")
	}
	t.Log(a, d)
}
