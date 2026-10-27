package body_test

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

/*
TestOnlyTheAddressPackageNarrows keeps every Address-to-cell crossing in
internal/body/address. //sp:narrows is what lets a function convert an
Address to a cell, so a second function carrying it is a second crossing, and
that one would not be checked.
*/
func TestOnlyTheAddressPackageNarrows(t *testing.T) {
	const allowed = "body/address/address.go"
	internal, err := os.OpenRoot("..")
	if err != nil {
		t.Fatalf("opening internal: %v", err)
	}
	defer func() { _ = internal.Close() }()
	tree := internal.FS()
	err = fs.WalkDir(tree, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.Contains(path, "spbody") {
			return err
		}
		src, err := fs.ReadFile(tree, path)
		if err != nil {
			return err
		}
		for line := range strings.Lines(string(src)) {
			if strings.TrimSpace(line) == "//sp:narrows" && path != allowed {
				t.Errorf("%s carries //sp:narrows; only %s may", path, allowed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal: %v", err)
	}
}
