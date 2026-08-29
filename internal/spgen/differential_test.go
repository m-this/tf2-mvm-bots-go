package spgen_test

import (
	"errors"
	"testing"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

func toolchain(t *testing.T) spshell.Toolchain {
	t.Helper()
	tc, err := spshell.ToolchainFromEnv()
	if err != nil {
		if errors.Is(err, spshell.ErrNoToolchain) {
			t.Skipf("no standalone SourcePawn toolchain: %v", err)
		}
		t.Fatal(err)
	}
	return tc
}

// TestGeneratedSourcePawnAgreesWithGo is the deliverable. Every combination
// the engine can produce goes through the generated SourcePawn, under
// SourcePawn's own VM, and through the Go it was generated from, and the two
// have to answer the same thing every time.
//
// Two answers per combination: the pure port, and the outcome the lazy table
// walks to. Nothing is sampled, so there is no fraction to argue about.
func TestGeneratedSourcePawnAgreesWithGo(t *testing.T) {
	tc := toolchain(t)
	if want := emit(t).Pure; !goldenMatches(t, goldenPure, want) {
		t.Fatal("testdata/actionsel.sp is stale: run go test ./internal/spgen -update")
	}

	start := time.Now()
	cells, err := tc.Run(t.Context(), "testdata/sweep.sp", nil)
	if err != nil {
		t.Fatalf("running the sweep under spshell: %v", err)
	}
	elapsed := time.Since(start)

	i, mismatches := 0, 0
	const reportAtMost = 10
	for p := range sweep {
		if i+2 > len(cells) {
			t.Fatalf("spshell answered %d cells, and the sweep is not finished at %s", len(cells), p)
		}
		want := int32(actionsel.Select(p.state, p.class, p.flags()))
		pure, walked := cells[i], cells[i+1]
		i += 2
		if pure == want && walked == want {
			continue
		}
		mismatches++
		if mismatches <= reportAtMost {
			t.Errorf("%s: Go says %d, the generated function says %d, the table walks to %d", p, want, pure, walked)
		}
	}
	if mismatches > reportAtMost {
		t.Errorf("%d further disagreements", mismatches-reportAtMost)
	}
	if i != len(cells) {
		t.Errorf("spshell answered %d cells, the sweep wanted %d", len(cells), i)
	}
	t.Logf("compared %d combinations, %d answers, in %s under spshell", i/2, i, elapsed.Round(time.Millisecond))
}

func goldenMatches(t *testing.T, path, want string) bool {
	t.Helper()
	got, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got == want
}

// goldenRow is the input the golden table is emitted from: the two values the
// edge already holds, then one column per predicate. Field order is the order
// the columns come out in, and testdata/golden.sp reads them by name.
type goldenRow struct {
	State              int32
	Class              int32
	MoneyToCollect     bool
	InUpgradeZone      bool
	ShoppedThisBreak   bool
	MovingToFront      bool
	UpgradesEnabled    bool
	HasUpgraded        bool
	UpgradeMidRound    bool
	HasSniperRifle     bool
	SniperStalled      bool
	AttackTargetFound  bool
	TankTargetFound    bool
	GiantToMark        bool
	NearbyMoney        bool
	StickyTrapPossible bool
}

func rowOf(p point) goldenRow {
	f := p.flags()
	return goldenRow{
		State: int32(p.state), Class: int32(p.class),
		MoneyToCollect: f.MoneyToCollect, InUpgradeZone: f.InUpgradeZone,
		ShoppedThisBreak: f.ShoppedThisBreak, MovingToFront: f.MovingToFront,
		UpgradesEnabled: f.UpgradesEnabled, HasUpgraded: f.HasUpgraded,
		UpgradeMidRound: f.UpgradeMidRound, HasSniperRifle: f.HasSniperRifle,
		SniperStalled: f.SniperStalled, AttackTargetFound: f.AttackTargetFound,
		TankTargetFound: f.TankTargetFound, GiantToMark: f.GiantToMark,
		NearbyMoney: f.NearbyMoney, StickyTrapPossible: f.StickyTrapPossible,
	}
}

// goldenStride picks every 512th reachable combination. The sweep above is the
// coverage argument; this one is the harness the bead asked for, and a table
// small enough that a failure names a row a person can read.
const goldenStride = 512

// TestGoldenTableRoundTrip takes the rows from a Go struct type, emits them as
// the SourcePawn golden table, and reads one result per row back.
func TestGoldenTableRoundTrip(t *testing.T) {
	tc := toolchain(t)

	var rows []goldenRow
	var points []point
	n := 0
	for p := range sweep {
		if n%goldenStride == 0 {
			rows = append(rows, rowOf(p))
			points = append(points, p)
		}
		n++
	}

	table, err := spshell.GoldenTable("gInputs", rows)
	if err != nil {
		t.Fatal(err)
	}
	cells, err := tc.Run(t.Context(), "testdata/golden.sp", map[string]string{"golden_inputs.inc": table})
	if err != nil {
		t.Fatalf("running the golden table under spshell: %v", err)
	}
	if len(cells) != 2*len(rows) {
		t.Fatalf("spshell answered %d cells for %d rows", len(cells), len(rows))
	}

	for i, p := range points {
		wantPort := int32(actionsel.Select(p.state, p.class, p.flags()))
		wantFilled := int32(actionsel.SelectFilled(p.state, p.class, p.flags()))
		if cells[2*i] != wantPort {
			t.Errorf("row %d, %s: Select is %d in SourcePawn, %d in Go", i, p, cells[2*i], wantPort)
		}
		if cells[2*i+1] != wantFilled {
			t.Errorf("row %d, %s: SelectFilled is %d in SourcePawn, %d in Go", i, p, cells[2*i+1], wantFilled)
		}
	}
	t.Logf("%d golden rows of %d fields, %d results read back", len(rows), len(spgen.ActionSelPredicates)+2, len(cells))
}

// TestGoldenTableRefusesAFieldWithNoCell keeps the harness honest about what a
// cell can hold, rather than emitting a plausible zero.
func TestGoldenTableRefusesAFieldWithNoCell(t *testing.T) {
	type row struct {
		Name  int32
		Count int64
	}
	if _, err := spshell.GoldenTable("gInputs", []row{{}}); err == nil {
		t.Fatal("an int64 field was accepted, and there is no cell for it")
	}
}
