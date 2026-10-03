package sdkcalls

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// CTFBot::SetMission writes m_mission, so a client that is not a CTFBot is never
// handed to it (mvm-km5). Nothing but the nextbot question is answered here, so
// the SDKCall fails the test by name if it is made.
func TestSetMissionIsNotMadeOnAPlainPlayer(t *testing.T) {
	t.Cleanup(engine.InstallBots(engine.BotCalls{
		NextBotOf: func(int32) engine.Bot { return 0 },
	}))
	t.Cleanup(engine.InstallSDKCalls(engine.SDKCallCalls{}))
	t.Cleanup(engine.InstallResets(engine.ResetCalls{}))

	SetMission(4, 2, true)
}

// And on a CTFBot it is made, after the behaviour it resets is forgotten.
func TestSetMissionIsMadeOnACTFBot(t *testing.T) {
	var order []string
	t.Cleanup(engine.InstallBots(engine.BotCalls{
		NextBotOf: func(int32) engine.Bot { return 0x1000 },
	}))
	t.Cleanup(engine.InstallSDKCalls(engine.SDKCallCalls{
		DoSetMission: func(int32, int32, bool) { order = append(order, "SetMission") },
	}))
	t.Cleanup(engine.InstallResets(engine.ResetCalls{
		ForgetMainActionStart: func(int32) { order = append(order, "Forget") },
	}))

	SetMission(7, 2, true)

	if len(order) != 2 || order[0] != "Forget" || order[1] != "SetMission" {
		t.Fatalf("calls were %v, want Forget then SetMission", order)
	}
}
