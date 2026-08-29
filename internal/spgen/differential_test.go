package spgen_test

import (
	"testing"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

// TestGeneratedSourcePawnAgreesWithGo is the deliverable. Every combination
// the engine can produce is walked through the generated table, under
// SourcePawn's own VM, and through the Go it was built from, and the two have
// to answer the same thing every time.
//
// It walks the table because the table is what ships: the plugin includes the
// data and the edge, and calls nothing else. Nothing is sampled, so there is
// no fraction to argue about.
func TestGeneratedSourcePawnAgreesWithGo(t *testing.T) {
	tc := spshell.ForTest(t)
	if want := emit(t).Data; !goldenMatches(t, goldenData, want) {
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
		if i >= len(cells) {
			t.Fatalf("spshell answered %d cells, and the sweep is not finished at %s", len(cells), p)
		}
		want := int32(actionsel.Select(p.state, p.class, p.flags()))
		walked := cells[i]
		i++
		if walked == want {
			continue
		}
		mismatches++
		if mismatches <= reportAtMost {
			t.Errorf("%s: Go says %d, the table walks to %d", p, want, walked)
		}
	}
	if mismatches > reportAtMost {
		t.Errorf("%d further disagreements", mismatches-reportAtMost)
	}
	if i != len(cells) {
		t.Errorf("spshell answered %d cells, the sweep wanted %d", len(cells), i)
	}
	t.Logf("compared %d combinations in %s under spshell", i, elapsed.Round(time.Millisecond))
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
	tc := spshell.ForTest(t)

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
	if len(cells) != len(rows) {
		t.Fatalf("spshell answered %d cells for %d rows", len(cells), len(rows))
	}

	for i, p := range points {
		if want := int32(actionsel.Select(p.state, p.class, p.flags())); cells[i] != want {
			t.Errorf("row %d, %s: the table walks to %d in SourcePawn, Select answers %d", i, p, cells[i], want)
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

// TestGeneratedEdgeCompiles compiles the edge against stubs of the plugin
// symbols it calls. The edge cannot run under spshell, because it is the one
// generated file that calls into the engine, but compiling it catches a
// reserved word used as a parameter name, a typo in a behaviour name, and a
// switch over an outcome the enum does not have.
func TestGeneratedEdgeCompiles(t *testing.T) {
	tc := spshell.ForTest(t)
	out := emit(t)
	if !goldenMatches(t, goldenData, out.Data) || !goldenMatches(t, goldenDispatch, out.Dispatch) {
		t.Fatal("the golden SourcePawn is stale: run go test ./internal/spgen -update")
	}
	if err := tc.Compile(t.Context(), "testdata/dispatch_smoke.sp", stubbedEnv); err != nil {
		t.Fatalf("compiling the generated edge: %v", err)
	}
}

/*
	stubbedEnv and sourceModEnv are the two halves of the smoke harness

The symbols SourceMod declares itself have to come from one place or the other
and never both: declaring them in the smoke file compiles under the standalone
compiler and collides under SourceMod's, which is why the edge was checked by
one compiler only.
*/
var stubbedEnv = map[string]string{"smoke_env.inc": `
enum Action
{
	Plugin_Continue = 0,
	Plugin_Handled = 3
};

enum RoundState
{
	RoundState_Init = 0
};

enum TFClassType
{
	TFClass_Unknown = 0
};

#define INVALID_ACTION 0
#define MAXPLAYERS 65

methodmap ConVar
{
	property bool BoolValue
	{
		public get() { return true; }
	}
}

/* SourceMod's own, needed by the generated attribute lookup. Comparing to the
   terminator rather than looping to it, so the semantics match: two names are
   equal when they end together with nothing different before it. */
stock bool StrEqual(const char[] a, const char[] b)
{
	int i = 0;
	while (a[i] != 0 && a[i] == b[i])
		i++;

	return a[i] == b[i];
}

stock RoundState GameRules_GetRoundState() { return RoundState_Init; }
stock TFClassType TF2_GetPlayerClass(int client) { return view_as<TFClassType>(client); }
`}

// sourceModEnv is the real headers. tf2_stocks rather than tf2, because
// TF2_GetPlayerClass is a stock there and the edge calls it.
var sourceModEnv = map[string]string{"smoke_env.inc": `
#include <sourcemod>
#include <sdktools>
#include <tf2_stocks>
#define INVALID_ACTION 0
`}
