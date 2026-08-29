package spgen_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestBothCompilersAgreeOnTheGeneratedTable

The generated source was checked by one compiler and shipped by another, which
means the check did not cover what ships. This walks the same sweep twice, once
through the 1.13 spcomp built beside spshell and once through the 1.12 spcomp64
the plugin builds with, and requires the two to answer the same cell every time.

Both runs use spshell's VM. Only the compiler differs, so a disagreement is the
compiler and nothing else.
*/
func TestBothCompilersAgreeOnTheGeneratedTable(t *testing.T) {
	local := spshell.ForTest(t)
	dir, err := upstream.Dir()
	if err != nil {
		t.Skipf("no plugin repository, set MVMBOTS_UPSTREAM: %v", err)
	}
	shipped, err := local.WithSourceMod(dir)
	if err != nil {
		t.Skipf("no SourceMod compiler: %v", err)
	}

	byLocal, err := local.Run(t.Context(), "testdata/sweep.sp", nil)
	if err != nil {
		t.Fatalf("sweep under the standalone spcomp: %v", err)
	}
	byShipped, err := shipped.Run(t.Context(), "testdata/sweep.sp", nil)
	if err != nil {
		t.Fatalf("sweep under SourceMod's spcomp64: %v", err)
	}

	if len(byLocal) != len(byShipped) {
		t.Fatalf("the two compilers produced %d and %d cells", len(byLocal), len(byShipped))
	}
	mismatches := 0
	const reportAtMost = 10
	for i := range byLocal {
		if byLocal[i] == byShipped[i] {
			continue
		}
		mismatches++
		if mismatches <= reportAtMost {
			t.Errorf("cell %d: spcomp says %d, spcomp64 says %d", i, byLocal[i], byShipped[i])
		}
	}
	if mismatches > reportAtMost {
		t.Errorf("%d cells disagree, %d not reported", mismatches, mismatches-reportAtMost)
	}
	t.Logf("%d cells, two compilers, no disagreement", len(byLocal))
}
