package actionsel

import (
	"fmt"
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/gosubset"
)

// TestSelectionIsInsideTheSubset holds the package to the subset the body
// generator translates, because the point of writing the choice here is that
// it becomes the SourcePawn it replaces.
func TestSelectionIsInsideTheSubset(t *testing.T) {
	diags, err := gosubset.CheckDir(".", gosubset.DefaultConfig())
	if err != nil {
		t.Fatalf("checking the package: %v", err)
	}
	if err := gosubset.Join(diags); err != nil {
		t.Fatal(err)
	}
}

// TestFlagBitsAreWhole fails when a flag is added to the domain and left out
// of the enumeration, which would silently shrink what exhaustiveness covers.
func TestFlagBitsAreWhole(t *testing.T) {
	if len(flagNames) != flagCount {
		t.Fatalf("flagNames has %d entries, want %d", len(flagNames), flagCount)
	}
	for i := range flagCount {
		set := setBits(flagBit(1 << uint32(i)))
		if len(set) != 1 || set[0] != i {
			t.Errorf("bit %d (%s) sets the fields %v", i, flagNames[i], set)
		}
	}
	if set := setBits(flagBit(1<<flagCount - 1)); len(set) != flagCount {
		t.Errorf("all bits set covers %d fields, want %d: a field was added without a bit", len(set), flagCount)
	}
}

// setBits reads a Flags back as the bit positions it has on, using flagBit's
// own inverse so the two stay a pair without reflection.
func setBits(f Flags) []int {
	on := []bool{
		f.MoneyToCollect, f.InUpgradeZone, f.ShoppedThisBreak, f.MovingToFront,
		f.UpgradesEnabled, f.HasUpgraded, f.UpgradeMidRound, f.HasSniperRifle,
		f.SniperStalled, f.AttackTargetFound, f.TankTargetFound, f.GiantToMark,
		f.NearbyMoney, f.StickyTrapPossible,
	}
	var set []int
	for i, v := range on {
		if v {
			set = append(set, i)
		}
	}
	return set
}

// TestEveryOutcomeIsNamed fails when an outcome is added to the enum and left
// out of the name table, which is the features.sp bug in a new place.
func TestEveryOutcomeIsNamed(t *testing.T) {
	for a := ActionNone; a <= ActionStrandedAsShipped; a++ {
		if _, named := actionNames[a]; !named {
			t.Errorf("the outcome %d has no name", int32(a))
		}
	}
	if len(actionNames) != int(ActionStrandedAsShipped)+1 {
		t.Errorf("actionNames has %d entries for %d outcomes", len(actionNames), int(ActionStrandedAsShipped)+1)
	}
}

// TestSuspendsSplitsTheEnum pins the order the dispatch depends on: every
// suspending outcome is below the first Plugin_Continue and every silence is
// above it.
func TestSuspendsSplitsTheEnum(t *testing.T) {
	if Suspends(ActionNone) {
		t.Error("ActionNone is not a behaviour to suspend for")
	}
	for a := ActionCollectMoneyIsPossible; a <= ActionGuardPoint; a++ {
		if !Suspends(a) {
			t.Errorf("%s should suspend", name(a))
		}
	}
	for a := ActionKeepWalkingToFront; a <= ActionStrandedAsShipped; a++ {
		if Suspends(a) {
			t.Errorf("%s is a Plugin_Continue and should not suspend", name(a))
		}
	}
}

// TestSelectIsTotal is the exhaustiveness assertion. Every reachable
// combination gets a named outcome, and a hole is reported as the combination
// that produced it.
func TestSelectIsTotal(t *testing.T) {
	assertTotal(t, "Select", Select)
}

func assertTotal(t *testing.T, who string, choose func(RoundState, Class, Flags) Action) {
	t.Helper()
	const reportAtMost = 10
	holes := 0
	for c := range reachable {
		a := choose(c.state, c.class, c.flags())
		if _, named := actionNames[a]; named && a != ActionNone {
			continue
		}
		holes++
		if holes <= reportAtMost {
			t.Errorf("%s has no outcome for %s: it answered %s", who, c, name(a))
		}
	}
	if holes > reportAtMost {
		t.Errorf("%s: %d further combinations with no outcome", who, holes-reportAtMost)
	}
}

// TestExhaustivenessCatchesAPunchedHole proves the assertion above can fail,
// and that it fails naming the combination rather than a count.
func TestExhaustivenessCatchesAPunchedHole(t *testing.T) {
	holed := func(state RoundState, class Class, f Flags) Action {
		a := Select(state, class, f)
		if state == RoundBetweenRounds && class == ClassSpy && a == ActionKeepOwnBreakBehaviour {
			return ActionNone
		}
		return a
	}

	fake := &testing.T{}
	assertTotal(fake, "holed", holed)
	if !fake.Failed() {
		t.Fatal("a punched hole did not fail the exhaustiveness assertion")
	}

	c := combination{state: RoundBetweenRounds, class: ClassSpy, bits: 1 << 2}
	if got := holed(c.state, c.class, c.flags()); got != ActionNone {
		t.Fatalf("%s answered %s, want the punched hole", c, name(got))
	}
	if want := "BetweenRounds / Spy / ShoppedThisBreak"; c.String() != want {
		t.Fatalf("the combination reads %q, want %q", c, want)
	}
}

// strandClass groups the stranded combinations by round state and class.
type strandClass struct {
	state RoundState
	class Class
}

func (s strandClass) String() string {
	return fmt.Sprintf("%s / %s", roundStateNames[s.state], classNames[s.class])
}

// TestStrandedCombinationsAreExactlyTheShippedOnes is the behaviour-equivalence
// claim for the port's worst case. The shipped chain leaves a bot with no
// behaviour on one class of input, mvm-vnn, and the port has to leave the same
// bots standing: no more, and no fewer. A fix that arrives with the port is a
// port bug, and so is a new hole.
func TestStrandedCombinationsAreExactlyTheShippedOnes(t *testing.T) {
	countOf := map[strandClass]int{}
	firstOf := map[strandClass]combination{}
	for c := range reachable {
		if Select(c.state, c.class, c.flags()) != ActionStrandedAsShipped {
			continue
		}
		k := strandClass{state: c.state, class: c.class}
		countOf[k]++
		if _, seen := firstOf[k]; !seen {
			firstOf[k] = c
		}
	}

	got := make([]string, 0, len(countOf))
	for k, n := range countOf {
		got = append(got, fmt.Sprintf("%s, %d combinations, for example %s", k, n, firstOf[k]))
	}
	slices.Sort(got)

	want := []string{
		"RoundRunning / Scout, 208 combinations, for example RoundRunning / Scout / no flags set",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the port strands bots in:\n%swant:\n%s", joinLines(got), joinLines(want))
	}
}

// TestSelectFilledDiffersOnlyOnTheStranded keeps the candidate fix off the
// port: SelectFilled may answer differently exactly where Select strands, and
// nowhere else.
func TestSelectFilledDiffersOnlyOnTheStranded(t *testing.T) {
	for c := range reachable {
		a, b := Select(c.state, c.class, c.flags()), SelectFilled(c.state, c.class, c.flags())
		switch {
		case a == ActionStrandedAsShipped:
			if b != ActionGuardPoint {
				t.Fatalf("%s: SelectFilled answers %s, want GuardPoint", c, name(b))
			}
		case a != b:
			t.Fatalf("%s: SelectFilled answers %s where the port answers %s", c, name(b), name(a))
		}
	}
}

// TestUnreachableCombinationsAreRefused pins the reachability rule, so that
// widening it is a decision rather than an accident.
func TestUnreachableCombinationsAreRefused(t *testing.T) {
	tests := []struct {
		name  string
		class Class
		flags Flags
		want  bool
	}{
		{name: "a sniper may hold a rifle", class: ClassSniper, flags: Flags{HasSniperRifle: true}, want: true},
		{name: "a sniper may be stalled", class: ClassSniper, flags: Flags{SniperStalled: true}, want: true},
		{name: "a heavy holds no rifle", class: ClassHeavy, flags: Flags{HasSniperRifle: true}, want: false},
		{name: "a heavy never stalls", class: ClassHeavy, flags: Flags{SniperStalled: true}, want: false},
		{name: "a heavy with neither", class: ClassHeavy, flags: Flags{}, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Reachable(tc.class, tc.flags); got != tc.want {
				t.Errorf("Reachable = %v, want %v", got, tc.want)
			}
		})
	}
}

func joinLines(ss []string) string {
	out := ""
	for _, s := range ss {
		out += "  " + s + "\n"
	}
	return out
}
