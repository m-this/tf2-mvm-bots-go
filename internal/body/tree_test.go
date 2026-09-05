package body_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/*
treeCopy is this repository's Go under a directory the test owns.

The two fixture tests below add a file to a live package and generate from the
working tree. internal/tables generates from that same tree, go test runs the
two packages at once, and whichever looked second saw the other's fixture:
either a per-client array nothing clears, or campbomb.sp carrying a function
that is not in the committed one. It failed make check twice in three runs
before this existed.

Only Go, and only what the generator reads: it type checks with go/types, so it
needs the module file and the sources, and nothing else in the tree is opened.
*/
func treeCopy(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for _, from := range []string{"go.mod", "go.sum"} {
		body, err := os.ReadFile(filepath.Join("../..", from))

		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			t.Fatalf("reading %s: %v", from, err)
		}

		// 0o600: a temporary tree this test is the only reader of.
		if err := os.WriteFile(filepath.Join(root, from), body, 0o600); err != nil {
			t.Fatalf("writing %s: %v", from, err)
		}
	}

	var sources []string

	// Collected first and copied after, rather than copied in the callback: a
	// read inside the walk is what gosec's G122 is about.
	err := filepath.WalkDir("../../internal", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			sources = append(sources, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the tree: %v", err)
	}

	for _, path := range sources {
		relative, err := filepath.Rel("../..", path)
		if err != nil {
			t.Fatalf("placing %s: %v", path, err)
		}

		target := filepath.Join(root, relative)

		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatalf("making %s: %v", filepath.Dir(target), err)
		}

		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		if err := os.WriteFile(target, body, 0o600); err != nil {
			t.Fatalf("writing %s: %v", target, err)
		}
	}

	return root
}
