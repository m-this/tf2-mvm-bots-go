package stocks

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
The ready command is what starts the wave, so it is still sent; what is bounded
is how often a refused one is sent again. ReadyDefender calls this every frame
while the flag disagrees, and the game refuses a ready for the first seconds
after a wave ends: without the bound every one of those frames spoke the
mercenary's Ready line.
*/
func TestARefusedReadyIsSentAgainOnceASecondAndNotEveryFrame(t *testing.T) {
	now := float32(100.0)
	nextTime := map[int32]float32{}
	commands := 0

	defer engine.InstallEntities(engine.EntityCalls{
		GameRulesPropAt: func(prop string, size int32, element int32) int32 {
			if prop != "m_bPlayerReady" || size != 1 || element != 7 {
				t.Fatalf("read %s size=%d element=%d", prop, size, element)
			}
			return 0 // The game refuses every one of these, as it does after a wave.
		},
	})()
	defer engine.InstallRegistrations(engine.RegisterCalls{
		NextReadyCommandTime:    func(client int32) float32 { return nextTime[client] },
		SetNextReadyCommandTime: func(client int32, when float32) { nextTime[client] = when },
	})()
	defer engine.InstallSpy(engine.SpyCalls{GameTime: func() float32 { return now }})()
	defer engine.InstallPlayers(engine.PlayerCalls{
		FakeClientCommand: func(_ int32, _ string, _ ...any) { commands++ },
	})()

	for range 66 { // One second of frames.
		SetPlayerReady(7, true)
		now += 1.0 / 66.0
	}
	if commands != 1 {
		t.Fatalf("a second of refused frames sent %d commands, want one", commands)
	}

	now += ReadyRetryInterval
	SetPlayerReady(7, true)
	if commands != 2 {
		t.Fatalf("the next second sent %d commands in total, want two", commands)
	}
	// A map change restarts the engine clock. A timestamp from the old map
	// must not silence F4 until the new clock catches up.
	now = 5
	SetPlayerReady(7, true)
	if commands != 3 {
		t.Fatalf("clock reset left ready blocked: got %d commands, want three", commands)
	}
	SetPlayerReady(7, true)
	if commands != 3 {
		t.Fatal("clock reset disabled retry throttling")
	}
}

// A bot already in the state it is being asked for presses nothing, and so
// spends none of its retry window on a command the game would ignore.
func TestAReadyAlreadyPressedIsNotPressedAgain(t *testing.T) {
	pressed := 0
	defer engine.InstallEntities(engine.EntityCalls{
		GameRulesPropAt: func(_ string, _ int32, _ int32) int32 { return 1 },
	})()
	defer engine.InstallPlayers(engine.PlayerCalls{
		FakeClientCommand: func(_ int32, _ string, _ ...any) { pressed++ },
	})()

	SetPlayerReady(8, true)
	if pressed != 0 {
		t.Fatalf("a bot that was already ready pressed ready %d times", pressed)
	}
}
