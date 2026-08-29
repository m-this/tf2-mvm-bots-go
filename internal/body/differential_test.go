package body_test

import (
	"math"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/roster"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

// goCells runs the Go the bodies were written as, in the order the probe runs
// them, and produces the same cell stream the probe prints.
func goCells(w world) []int32 {
	in := &installed{w: w}
	defer engine.Install(in.calls())()

	roster.ResetState()
	for slot := int32(1); slot <= worldSlots; slot++ {
		roster.SetDefenderBot(slot, w.defender[slot])
	}

	var out []int32
	emit := func(result int32) {
		out = append(out, result, int32(len(in.trace))) //nolint:gosec // G115: record bounds the trace at traceCells
		out = append(out, in.trace...)
		in.trace = in.trace[:0]
	}
	for team := int32(0); team < 4; team++ {
		emit(roster.AliveOnTeam(worldSlots, team))
	}
	for team := int32(0); team < 4; team++ {
		centre := roster.TeamCentre(worldSlots, team)
		for _, axis := range centre {
			// The bit pattern, not a rounded number: two floats that
			// print the same and are not the same is the failure
			// this comparison exists to catch.
			out = append(out, int32(math.Float32bits(axis))) //nolint:gosec // G115: a cell is 32 bits either way
		}
		emit(0)
	}
	for weapon := int32(0); weapon <= worldSlots; weapon++ {
		emit(roster.LoadedRounds(weapon))
	}
	// c is the shift amount and stays at or above zero: a negative one is
	// undefined in SourcePawn and a panic in Go, so it is not a case where
	// the two could be compared at all.
	for a := int32(-3); a <= 3; a++ {
		for b := int32(-3); b <= 3; b++ {
			for c := int32(0); c <= 3; c++ {
				emit(roster.Shifted(a, b, c))
				emit(roster.Ored(a, b, c))
				emit(roster.Mixed(a, b, c))
				emit(roster.Chained(a, b, c))
			}
		}
	}
	for player := int32(1); player <= worldSlots; player++ {
		supercede, _ := roster.IsBotPre(player)
		emit(boolCell(supercede))

		roster.MyTouchPre(0, player)
		supercede, value := roster.IsBotPre(player)
		emit(boolCell(supercede))
		out = append(out, boolCell(value))

		roster.MyTouchPost(0, player)
		supercede, _ = roster.IsBotPre(player)
		emit(boolCell(supercede))
	}
	return out
}

/*
	TestGeneratedBodiesAgreeWithTheGoTheyCameFrom

The deliverable of mvm-bis. A body that calls the engine cannot be run here, so
the engine is a stub on both sides answering out of one canned world, and what
is compared is the answer and the sequence of calls that produced it.

Nothing is sampled. Every case the probe runs is compared cell for cell.
*/
func TestGeneratedBodiesAgreeWithTheGoTheyCameFrom(t *testing.T) {
	tc := spshell.ForTest(t)
	w := cannedWorld()

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	cells, err := tc.Run(t.Context(), "testdata/roster_probe.sp", map[string]string{
		"roster.sp":        string(generated["sourcepawn/roster.sp"]),
		"roster_world.inc": worldInclude(w),
	})
	if err != nil {
		t.Fatalf("running the probe under spshell: %v", err)
	}

	compareCells(t, goCells(w), cells)
}
