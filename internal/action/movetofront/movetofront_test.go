package movetofront

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
A rally order lapses against the engine clock, so asking twice inside one tick
could once give two answers: the wave-started check would decline to end the
action because the order was live, and SetPlayerReady would run a few lines
later because it no longer was. The mode is latched on entry instead, and the
test for that is that the walk asks the engine exactly once however often it
reads the answer.
*/
func TestTheRallyModeIsSettledOnceForTheWholeTick(t *testing.T) {
	asked := 0
	restoreStations := engine.InstallStations(engine.StationCalls{
		RoundState: engine.RoundStateRunning,
	})
	defer restoreStations()
	restoreDirectives := engine.InstallDirectives(engine.DirectiveCalls{
		RallyActive: func(int32) bool {
			asked++
			// The order lapses the instant it is first read.
			return asked == 1
		},
	})
	defer restoreDirectives()

	const actor = 3
	latchRallyWalk(actor)
	for range 5 {
		if !directiveRallyWalk(actor) {
			t.Fatal("the tick changed its mind about walking to the rally point")
		}
	}
	if asked != 1 {
		t.Errorf("the engine was asked %d times in one tick, want 1", asked)
	}
}

// The other half: no order means the ordinary walk, and it stays that way.
func TestNoRallyOrderLeavesTheOrdinaryWalk(t *testing.T) {
	restoreStations := engine.InstallStations(engine.StationCalls{
		RoundState: engine.RoundStateRunning,
	})
	defer restoreStations()
	restoreDirectives := engine.InstallDirectives(engine.DirectiveCalls{
		RallyActive: func(int32) bool { return false },
	})
	defer restoreDirectives()

	const actor = 4
	latchRallyWalk(actor)
	if directiveRallyWalk(actor) {
		t.Fatal("a bot with no order was sent to a rally point")
	}
}

// Between rounds is the shopping phase, which the rally order does not own
// however live it is: the caller's orders are for a running wave.
func TestARallyOrderIsIgnoredBetweenRounds(t *testing.T) {
	restoreStations := engine.InstallStations(engine.StationCalls{
		RoundState: engine.RoundStateBetweenRounds,
	})
	defer restoreStations()
	restoreDirectives := engine.InstallDirectives(engine.DirectiveCalls{
		RallyActive: func(int32) bool { return true },
	})
	defer restoreDirectives()

	const actor = 5
	latchRallyWalk(actor)
	if directiveRallyWalk(actor) {
		t.Fatal("a rally order took over the walk before the wave started")
	}
}
