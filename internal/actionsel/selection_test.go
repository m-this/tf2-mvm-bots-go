package actionsel

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/gosubset"
)

// TestSelectionIsInsideTheSubset holds the file to the subset the body
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
	typ := reflect.TypeOf(Flags{})
	if typ.NumField() != flagCount {
		t.Fatalf("Flags has %d fields and the enumeration covers %d; add the field to flagBit and flagNames", typ.NumField(), flagCount)
	}
	if len(flagNames) != flagCount {
		t.Fatalf("flagNames has %d entries, want %d", len(flagNames), flagCount)
	}
	for i := range flagCount {
		want := typ.Field(i).Name
		if flagNames[i] != want {
			t.Errorf("bit %d is named %q, want %q", i, flagNames[i], want)
		}
		v := reflect.ValueOf(flagBit(1 << uint32(i)))
		for j := range flagCount {
			set := v.Field(j).Bool()
			if set != (i == j) {
				t.Errorf("bit %d sets %s to %v", i, typ.Field(j).Name, set)
			}
		}
	}
}

// TestSelectIsTotal is the exhaustiveness assertion. Every reachable
// combination gets a named action, and a hole is reported as the combination
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
		if a != ActionNone {
			if _, named := actionNames[a]; named {
				continue
			}
		}
		holes++
		if holes <= reportAtMost {
			t.Errorf("%s has no action for %s: it answered %s", who, c, name(a))
		}
	}
	if holes > reportAtMost {
		t.Errorf("%s: %d further combinations with no action", who, holes-reportAtMost)
	}
}

// TestExhaustivenessCatchesAPunchedHole proves the assertion above can fail,
// and that it fails naming the combination rather than a count. holed is
// Select with the between-rounds answer for a spy removed, which is the shape
// of mvm-7kr.
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

	// The same hole, named, is what the failure message carries.
	c := combination{state: RoundBetweenRounds, class: ClassSpy, bits: 1 << 2}
	if got := holed(c.state, c.class, c.flags()); got != ActionNone {
		t.Fatalf("%s answered %s, want the punched hole", c, name(got))
	}
	if want := "BetweenRounds / Spy / ShoppedThisBreak"; c.String() != want {
		t.Fatalf("the combination reads %q, want %q", c, want)
	}
}

// holeClass groups the combinations where the shipped SourcePawn hands out no
// behaviour, by what the total function says the answer should have been.
type holeClass struct {
	state  RoundState
	class  Class
	answer Action
}

func (h holeClass) String() string {
	return fmt.Sprintf("%s / %s -> %s", roundStateNames[h.state], classNames[h.class], name(h.answer))
}

// TestShippedHoles is the cross-check against nextbot_behavior.sp as it
// stands. Every combination where GetDesiredBotAction returns Plugin_Continue
// with nothing on the stack is grouped, and the group is either a decision
// with a name or a bot left standing. The want list is the whole of it, so a
// new hole and a closed hole both fail here.
func TestShippedHoles(t *testing.T) {
	firstOf := map[holeClass]combination{}
	countOf := map[holeClass]int{}
	for c := range reachable {
		if Shipped(c.state, c.class, c.flags()) != ActionNone {
			continue
		}
		h := holeClass{state: c.state, class: c.class, answer: Select(c.state, c.class, c.flags())}
		countOf[h]++
		if _, seen := firstOf[h]; !seen {
			firstOf[h] = c
		}
	}

	var stranded []string
	for h := range countOf {
		if !deliberateNothing[h.answer] {
			stranded = append(stranded, fmt.Sprintf("%s, %d combinations, for example %s", h, countOf[h], firstOf[h]))
		}
	}
	slices.Sort(stranded)

	want := []string{
		"RoundRunning / Scout -> GuardPoint, 208 combinations, for example RoundRunning / Scout / no flags set",
	}
	if !slices.Equal(stranded, want) {
		t.Errorf("the shipped code strands bots in:\n%swant:\n%s", joinLines(stranded), joinLines(want))
	}
}

// TestShippedDeliberateNothing pins the places the shipped code says nothing
// on purpose. Each is a Keep action in the total function, and each is an
// assumption about somebody else having already given the bot a behaviour.
func TestShippedDeliberateNothing(t *testing.T) {
	seen := map[string]bool{}
	for c := range reachable {
		if Shipped(c.state, c.class, c.flags()) != ActionNone {
			continue
		}
		a := Select(c.state, c.class, c.flags())
		if deliberateNothing[a] {
			seen[name(a)] = true
		}
	}
	want := []string{
		"KeepHealing", "KeepOwnBreakBehaviour", "KeepSnipingPosition",
		"KeepWaitingForClass", "KeepWalkingToFront", "WaitOutsideRound",
	}
	got := slices.Sorted(maps.Keys(seen))
	if !slices.Equal(got, want) {
		t.Errorf("the deliberate silences are %v, want %v", got, want)
	}
}

// TestSelectAgreesWithShippedWhereShippedAnswers keeps the total function
// honest: it is the shipped choice plus the holes filled, and nothing else.
func TestSelectAgreesWithShippedWhereShippedAnswers(t *testing.T) {
	for c := range reachable {
		want := Shipped(c.state, c.class, c.flags())
		if want == ActionNone {
			continue
		}
		if got := Select(c.state, c.class, c.flags()); got != want {
			t.Fatalf("%s: Select says %s, the plugin says %s", c, name(got), name(want))
		}
	}
}

// TestClosedBugs is the four beads the epic names, each as the combination
// that produced it.
func TestClosedBugs(t *testing.T) {
	tests := []struct {
		name        string
		bead        string
		state       RoundState
		class       Class
		flags       Flags
		wantShipped Action
		wantSelect  Action
	}{
		{
			name:        "an engineer who already shopped is left to his nest",
			bead:        "mvm-7kr",
			state:       RoundBetweenRounds,
			class:       ClassEngineer,
			flags:       Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			wantShipped: ActionNone,
			wantSelect:  ActionKeepOwnBreakBehaviour,
		},
		{
			name:        "an engineer who has not shopped goes shopping",
			bead:        "mvm-7kr",
			state:       RoundBetweenRounds,
			class:       ClassEngineer,
			flags:       Flags{UpgradesEnabled: true},
			wantShipped: ActionGotoUpgrade,
			wantSelect:  ActionGotoUpgrade,
		},
		{
			name:        "a rifle sniper is refused the front and left to his perch",
			bead:        "mvm-pvt",
			state:       RoundBetweenRounds,
			class:       ClassSniper,
			flags:       Flags{ShoppedThisBreak: true, UpgradesEnabled: true, HasSniperRifle: true},
			wantShipped: ActionNone,
			wantSelect:  ActionKeepOwnBreakBehaviour,
		},
		{
			name:        "a sniper with no rifle walks to the front",
			bead:        "mvm-pvt",
			state:       RoundBetweenRounds,
			class:       ClassSniper,
			flags:       Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			wantShipped: ActionMoveToFront,
			wantSelect:  ActionMoveToFront,
		},
		{
			name:        "the medic follows his patient to the front",
			bead:        "mvm-e4g",
			state:       RoundBetweenRounds,
			class:       ClassMedic,
			flags:       Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			wantShipped: ActionMoveToFront,
			wantSelect:  ActionMoveToFront,
		},
		{
			name:        "a stalled sniper fights like one who never had a rifle",
			bead:        "mvm-489",
			state:       RoundRunning,
			class:       ClassSniper,
			flags:       Flags{HasUpgraded: true, UpgradesEnabled: true, HasSniperRifle: true, SniperStalled: true},
			wantShipped: ActionDefenderAttack,
			wantSelect:  ActionDefenderAttack,
		},
		{
			name:        "a sniper who kept his mission is left sniping",
			bead:        "mvm-489",
			state:       RoundRunning,
			class:       ClassSniper,
			flags:       Flags{HasUpgraded: true, UpgradesEnabled: true, HasSniperRifle: true},
			wantShipped: ActionNone,
			wantSelect:  ActionKeepSnipingPosition,
		},
		{
			name:        "a stalled sniper is sent to the front in the break",
			bead:        "mvm-489",
			state:       RoundBetweenRounds,
			class:       ClassSniper,
			flags:       Flags{ShoppedThisBreak: true, UpgradesEnabled: true, HasSniperRifle: true, SniperStalled: true},
			wantShipped: ActionMoveToFront,
			wantSelect:  ActionMoveToFront,
		},
		{
			name:        "a scout with no money, no giant and no target is stranded today",
			bead:        "the hole still shipping",
			state:       RoundRunning,
			class:       ClassScout,
			flags:       Flags{HasUpgraded: true, UpgradesEnabled: true},
			wantShipped: ActionNone,
			wantSelect:  ActionGuardPoint,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Shipped(tc.state, tc.class, tc.flags); got != tc.wantShipped {
				t.Errorf("%s: the plugin answers %s, want %s", tc.bead, name(got), name(tc.wantShipped))
			}
			if got := Select(tc.state, tc.class, tc.flags); got != tc.wantSelect {
				t.Errorf("%s: Select answers %s, want %s", tc.bead, name(got), name(tc.wantSelect))
			}
		})
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
