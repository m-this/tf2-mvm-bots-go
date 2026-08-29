package threat_test

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tf"
	"github.com/m-this/tf2-mvm-bots-go/internal/threat"
)

// TestPriorityIsTotal walks the whole domain and requires every combination to
// answer with a declared priority. A hole here would be a bot with no opinion
// about a target it can see.
func TestPriorityIsTotal(t *testing.T) {
	t.Parallel()

	declared := threat.Priorities()
	seen := map[threat.Priority]int{}
	walked := 0
	for got := range threat.Threats {
		p := threat.PriorityOf(got)
		if !slices.Contains(declared, p) {
			t.Fatalf("%+v answers %d, which is not a declared priority", got, int32(p))
		}
		seen[p]++
		walked++
	}
	if walked != threat.DomainSize {
		t.Fatalf("walked %d combinations, the domain is %d", walked, threat.DomainSize)
	}

	for _, p := range declared {
		if seen[p] == 0 {
			t.Errorf("%s is declared and nothing reaches it", p)
		}
	}
	t.Logf("%d combinations, %d priorities all reached", walked, len(declared))
}

/*
	TestUrgentOutranksEverything

The one rule the comment above the shipped function states outright: anything
inside THREAT_URGENT_RANGE outranks the list, because a priority target is worth
nothing to a corpse. It is a rule about the ordering as well as about the
answer, so both are checked.
*/
func TestUrgentOutranksEverything(t *testing.T) {
	t.Parallel()

	for got := range threat.Threats {
		if got.RangeSquared >= threat.UrgentRange*threat.UrgentRange {
			continue
		}
		if p := threat.PriorityOf(got); p != threat.PriorityUrgent {
			t.Fatalf("%+v is inside the urgent range and answers %s", got, p)
		}
	}
	for _, p := range threat.Priorities() {
		if p != threat.PriorityUrgent && p >= threat.PriorityUrgent {
			t.Errorf("%s is not urgent and does not rank below it", p)
		}
	}
}

/*
	TestTheBoundariesAreTheShippedOnes

The two comparisons are strict, so a range exactly on a boundary belongs to the
band above it. A port that wrote <= or >= would pass every other test in this
file and differ only here.
*/
func TestTheBoundariesAreTheShippedOnes(t *testing.T) {
	t.Parallel()

	player := func(rangeSq float32) threat.Threat {
		return threat.Threat{RangeSquared: rangeSq, IsPlayer: true, InGame: true, Class: tf.ClassMedic}
	}
	urgent := threat.UrgentRange * threat.UrgentRange
	far := threat.PriorityRange * threat.PriorityRange

	cases := []struct {
		name    string
		rangeSq float32
		want    threat.Priority
	}{
		{"a hair inside the urgent range", urgent - 1, threat.PriorityUrgent},
		{"exactly on the urgent range is not urgent", urgent, threat.PriorityMedic},
		{"exactly on the priority range is still a target", far, threat.PriorityMedic},
		{"a hair past the priority range is not", far + 1, threat.PriorityNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := threat.PriorityOf(player(tc.rangeSq)); got != tc.want {
				t.Errorf("at %g the answer is %s, want %s", tc.rangeSq, got, tc.want)
			}
		})
	}
}

/*
	TestTheTankStaysInvisible

mvm-z83.41: the port carries the defects it found. A tank is not a player, so
past the urgent range it answers PriorityNone whatever else is true of it, and
inside the urgent range it answers PriorityUrgent for the same reason anything
does. Neither is fixed here.

mvm-ds3 is the other half, and it is not in this function: the scans that find
threats walk player slots, so a tank is never handed to this decision at all.
The record shape is what makes fixing that possible, and taking the record is
the whole of what this bead changes.
*/
func TestTheTankStaysInvisible(t *testing.T) {
	t.Parallel()

	tank := threat.Threat{RangeSquared: threat.PriorityRange * threat.PriorityRange, Giant: true}
	if got := threat.PriorityOf(tank); got != threat.PriorityNone {
		t.Errorf("a distant tank answers %s, and the shipped code answers None", got)
	}

	tank.RangeSquared = 0
	if got := threat.PriorityOf(tank); got != threat.PriorityUrgent {
		t.Errorf("a tank in the bot's face answers %s, and the shipped code answers Urgent", got)
	}
}

// TestAnOutOfGamePlayerIsNotATarget covers the second half of the pair the
// shipped code tests together. Either being false is enough.
func TestAnOutOfGamePlayerIsNotATarget(t *testing.T) {
	t.Parallel()

	mid := (threat.UrgentRange*threat.UrgentRange + threat.PriorityRange*threat.PriorityRange) / 2
	for _, tc := range []struct{ isPlayer, inGame bool }{{false, true}, {true, false}, {false, false}} {
		got := threat.PriorityOf(threat.Threat{
			RangeSquared: mid, IsPlayer: tc.isPlayer, InGame: tc.inGame,
			Class: tf.ClassMedic, Giant: true, Carrier: true,
		})
		if got != threat.PriorityNone {
			t.Errorf("isPlayer=%t inGame=%t answers %s, want None", tc.isPlayer, tc.inGame, got)
		}
	}
}

/*
	TestGiantAndCarrierAreDeadForANonPlayer

The property the edge needs, and it is not a tidiness claim: it is what lets the
edge pass false rather than call for them.

The shipped chain reads TF2_IsMiniBoss and TF2_HasTheFlag only after the player
test has passed, so for anything that is not an in-game player it never asks.
An edge that fills the record eagerly does ask, and both throw: measured, 3933
exceptions in four waves, on tank_boss and obj_attachment_sapper. See
mvm-z83.46.

So the edge guards them, and this says the guard changes no answer.
*/
func TestGiantAndCarrierAreDeadForANonPlayer(t *testing.T) {
	t.Parallel()

	for got := range threat.Threats {
		if got.IsPlayer && got.InGame {
			continue
		}
		blank := got
		blank.Giant, blank.Carrier = false, false
		if want, guarded := threat.PriorityOf(got), threat.PriorityOf(blank); want != guarded {
			t.Fatalf("%+v answers %s, and with giant and carrier zeroed it answers %s", got, want, guarded)
		}
	}
}

/*
	TestClassIsDeadForANonPlayer

The same for the class, which the edge also cannot ask for: TF2_GetPlayerClass
on a non-player is the same kind of read.
*/
func TestClassIsDeadForANonPlayer(t *testing.T) {
	t.Parallel()

	for got := range threat.Threats {
		if got.IsPlayer && got.InGame {
			continue
		}
		blank := got
		blank.Class = tf.ClassUnknown
		if want, guarded := threat.PriorityOf(got), threat.PriorityOf(blank); want != guarded {
			t.Fatalf("%+v answers %s, and with the class blanked it answers %s", got, want, guarded)
		}
	}
}
