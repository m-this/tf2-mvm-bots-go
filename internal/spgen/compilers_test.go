package spgen_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
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
	shipped, err := local.WithSourceMod(upstream.SkipOrFail(t))
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

/*
	TestTheGeneratedEdgeCompilesUnderBothCompilers

The table above is data. The edge is the file that calls into the plugin, and it
is what a typo in a behaviour name or a switch over a missing outcome would show
up in, so checking it under one compiler and shipping it with the other left the
part that can actually be wrong uncovered.

It cannot run: it calls the engine. Compiling is the whole check.
*/
func TestTheGeneratedEdgeCompilesUnderBothCompilers(t *testing.T) {
	local := spshell.ForTest(t)
	shipped, err := local.WithSourceMod(upstream.SkipOrFail(t))
	if err != nil {
		t.Skipf("no SourceMod compiler: %v", err)
	}
	if err := shipped.Compile(t.Context(), "testdata/dispatch_smoke.sp", sourceModEnv); err != nil {
		t.Fatalf("compiling the generated edge with SourceMod's spcomp64: %v", err)
	}
}

/*
	TestGeneratedAttributeLookupAnswersTheDeclaredIDs

The attribute table is the one generated file whose whole job is a name to id
map, so the check is that the map it emits is the map the Go declared. Every
name goes through AttributeID under SourcePawn's own VM, plus a name the table
does not hold and an empty one, which both have to come back ATTRIBUTE_NONE.
*/
func TestGeneratedAttributeLookupAnswersTheDeclaredIDs(t *testing.T) {
	tc := spshell.ForTest(t)

	var names strings.Builder
	names.WriteString("char gProbeNames[][] =\n{\n")
	for _, a := range tables.Attributes {
		fmt.Fprintf(&names, "\t%q,\n", a.Name)
	}
	// The two misses: a name the schema could have and the table does not
	// hold, and the empty string, which a caller with no attribute passes.
	names.WriteString("\t\"a name the schema has and the ranking does not\",\n\t\"\",\n};\n")

	cells, err := tc.Run(t.Context(), "testdata/attributes_smoke.sp", map[string]string{
		"smoke_env.inc":   stubbedEnv["smoke_env.inc"],
		"probe_names.inc": names.String(),
		"attributes.sp":   string(tables.SourcePawnAttributes()),
	})
	if err != nil {
		t.Fatalf("running the attribute lookup: %v", err)
	}
	if want := len(tables.Attributes) + 2; len(cells) != want {
		t.Fatalf("%d answers for %d names plus the two misses", len(cells), want)
	}
	for i, a := range tables.Attributes {
		if cells[i] != a.ID {
			t.Errorf("%q looked up as %d, the table declares %d", a.Name, cells[i], a.ID)
		}
	}
	for i, cell := range cells[len(tables.Attributes):] {
		if cell != 0 {
			t.Errorf("miss %d answered %d, want ATTRIBUTE_NONE", i, cell)
		}
	}
}
