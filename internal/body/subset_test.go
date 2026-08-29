package body_test

import (
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/gosubset"
)

// TestEveryBodyIsInsideTheSubset is the first gate. internal/spbody will refuse
// what it cannot translate, but it refuses in its own words at emission time;
// gosubset refuses in the author's words with the fix beside it, and it is the
// checker SUBSET.md documents. A body that passes here translates.
func TestEveryBodyIsInsideTheSubset(t *testing.T) {
	cfg, err := body.SubsetConfig("../..")
	if err != nil {
		t.Fatalf("reading the extern declarations: %v", err)
	}
	for _, b := range body.All {
		t.Run(b.Dir, func(t *testing.T) {
			diags, err := gosubset.CheckDir(filepath.Join("../..", b.Dir), cfg)
			if err != nil {
				t.Fatalf("checking %s: %v", b.Dir, err)
			}
			if err := gosubset.Join(diags); err != nil {
				t.Error(err)
			}
		})
	}
}
