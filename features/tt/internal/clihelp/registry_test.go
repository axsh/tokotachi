package clihelp

import (
	"testing"
)

func TestRegisterLookup(t *testing.T) {
	path := "__test_register_path__"
	p := samplePage()
	p.Title = "tt test"
	Register(path, p)
	got, ok := Lookup(path)
	if !ok {
		t.Fatal("lookup miss")
	}
	if got.Title != "tt test" {
		t.Fatalf("title=%q", got.Title)
	}
	found := false
	for _, k := range AllPaths() {
		if k == path {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("AllPaths missing key")
	}
}
