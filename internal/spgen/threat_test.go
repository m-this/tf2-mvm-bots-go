package spgen_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
	"github.com/m-this/tf2-mvm-bots-go/internal/threat"
)

// threatEnv is the generated threat file plus the probe ranges, injected the
// same way the attribute lookup's names are.
func threatEnv(t *testing.T) map[string]string {
	t.Helper()

	var ranges strings.Builder
	ranges.WriteString("int gProbeRanges[] =\n{\n")
	for _, rangeSq := range threat.Ranges() {
		// The bits, not the decimal: the sweep compares the exact float32 the
		// Go compared, and view_as<float> reads them straight back.
		fmt.Fprintf(&ranges, "\t%d,\n", int32(math.Float32bits(rangeSq))) //nolint:gosec // G115: a cell is 32 bits either way
	}
	ranges.WriteString("};\n")

	return map[string]string{
		"smoke_env.inc":      stubbedEnv["smoke_env.inc"],
		"probe_ranges.inc":   ranges.String(),
		"threat_priority.sp": string(spgen.EmitThreatPriority()),
	}
}

/*
	TestGeneratedThreatPriorityAgreesWithGo

The deliverable of mvm-z83.6. Every combination the decision can be asked about
goes through the generated SourcePawn under SourcePawn's own VM and through the
Go it was generated from, and the two have to answer the same thing.

Nothing is sampled. The domain is the two range boundaries with a point either
side, both answers to each of the four booleans and every class, because the
decision reads a distance through exactly two comparisons and there is nothing
else a range can change.
*/
func TestGeneratedThreatPriorityAgreesWithGo(t *testing.T) {
	tc := spshell.ForTest(t)

	cells, err := tc.Run(t.Context(), "testdata/threat_smoke.sp", threatEnv(t))
	if err != nil {
		t.Fatalf("running the threat sweep under spshell: %v", err)
	}
	if len(cells) != threat.DomainSize {
		t.Fatalf("spshell answered %d cells for a domain of %d", len(cells), threat.DomainSize)
	}

	i, mismatches := 0, 0
	const reportAtMost = 10
	for got := range threat.Threats {
		want := int32(threat.PriorityOf(got))
		if cells[i] != want {
			mismatches++
			if mismatches <= reportAtMost {
				t.Errorf("%+v: the table answers %d, the Go answers %d", got, cells[i], want)
			}
		}
		i++
	}
	if mismatches > reportAtMost {
		t.Errorf("%d further combinations disagree", mismatches-reportAtMost)
	}
	t.Logf("compared %d combinations under spshell", i)
}

// TestThreatPriorityCompilesUnderBothCompilers puts the generated file through
// the compiler the plugin ships with as well, per mvm-z83.13.
func TestThreatPriorityCompilesUnderBothCompilers(t *testing.T) {
	local := spshell.ForTest(t)
	shipped, err := local.WithSourceMod(plugin.SkipOrFail(t))
	if err != nil {
		t.Skipf("no SourceMod compiler: %v", err)
	}

	env := threatEnv(t)
	env["smoke_env.inc"] = sourceModEnv["smoke_env.inc"]
	if err := shipped.Compile(t.Context(), "testdata/threat_smoke.sp", env); err != nil {
		t.Fatalf("compiling the generated threat table with SourceMod's spcomp64: %v", err)
	}
}
