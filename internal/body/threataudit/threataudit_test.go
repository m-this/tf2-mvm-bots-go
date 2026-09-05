package threataudit_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body/threataudit"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// tank is an entity index past every client slot, which is what a tank_boss
// is: IsClientInGame throws on it, and so does everything that reads a class.
const tank int32 = 70

// The Go ranges are zero, so a range of zero is inside neither band and the
// player test is reached.

// A record filled for a tank asks nothing a tank cannot answer.
//
// Only IsPlayer and the table are installed. Fill puts a panic naming itself
// behind every other call, so IsClientInGame, PlayerClass, IsMiniBoss or
// HasTheFlag reached for a non-player fails here rather than throwing on a
// server, which is mvm-z83.46.
func TestFillingATankAsksNothingOfAClient(t *testing.T) {
	defer engine.Install(engine.Calls{
		IsPlayer: func(entity int32) bool { return entity != tank },
	})()

	var asked struct {
		isPlayer, inGame bool
		class            engine.Class
	}
	defer engine.InstallThreatAudits(engine.ThreatAuditCalls{
		ThreatPriorityOf: func(_ float32, isPlayer bool, inGame bool, class engine.Class, _ bool, _ bool) int32 {
			asked.isPlayer, asked.inGame, asked.class = isPlayer, inGame, class
			return 7
		},
	})()

	if got := threataudit.ThreatPriorityGenerated(tank, 0); got != 7 {
		t.Fatalf("the table's answer was %d, want 7", got)
	}
	if asked.isPlayer || asked.inGame || asked.class != engine.ClassUnknown() {
		t.Errorf("the record for a tank was player=%v inGame=%v class=%d; want false, false, unknown",
			asked.isPlayer, asked.inGame, asked.class)
	}
}

// The shipped chain short-circuits the same way, and the port keeps that.
func TestTheChainAsksNothingOfATank(t *testing.T) {
	defer engine.Install(engine.Calls{
		IsPlayer: func(entity int32) bool { return entity != tank },
	})()

	if got := threataudit.ThreatPriority(tank, 0); got != engine.ThreatPriorityNone() {
		t.Fatalf("the chain ranked a tank %d, want none", got)
	}
}
