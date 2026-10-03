package offsets

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

const (
	plainPlayer = 4 // a CreateFakeClient body: a CTFPlayer with no nextbot
	gameBot     = 7 // tf_bot_add's, or the popfile's: a CTFBot
)

// installWorld answers for two fake clients, one of each kind. The field reads
// and writes are recorded rather than answered blind, so a test can say which
// client was touched.
func installWorld(t *testing.T) (touched *[]int32) {
	t.Helper()

	touched = &[]int32{}
	t.Cleanup(engine.Install(engine.Calls{
		MaxClients:     func() int32 { return 64 },
		IsClientInGame: func(int32) bool { return true },
	}))
	t.Cleanup(engine.InstallBots(engine.BotCalls{
		NextBotOf: func(entity int32) engine.Bot {
			if entity == gameBot {
				return 0x1000
			}
			return 0
		},
	}))
	t.Cleanup(engine.InstallOffsets(engine.OffsetCalls{
		FormatInto: func(string, []any) engine.Text { return engine.Text{} },
		OffsetMap:  func() engine.Properties { return 1 },
		ValueAt:    func(engine.Properties, engine.Text) (bool, int32) { return true, 0x289c },
		EntDataDefault: func(entity int32, _ int32) int32 {
			*touched = append(*touched, entity)
			return 7 // CTFBot::MISSION_DESTROY_SENTRIES, as the game numbers it
		},
		SetEntDataSized: func(entity int32, _ int32, _ engine.Cell, _ int32) {
			*touched = append(*touched, entity)
		},
		EntityAddress: func(entity int32) engine.Address {
			*touched = append(*touched, entity)
			return 0x2000
		},
	}))
	return touched
}

/*
A fake client is not always a CTFBot, and a CTFBot field read off a CTFPlayer
runs past the end of it: apw-4ei has the server faulting reading m_mission at
0x289c off one (mvm-km5). None of the three fields is touched on a client that
has no nextbot.
*/
func TestNoCTFBotFieldIsTouchedOnAPlainPlayer(t *testing.T) {
	touched := installWorld(t)

	if mission := GetTFBotMission(plainPlayer); mission != 0 {
		t.Errorf("GetTFBotMission on a plain player = %d, want 0, no mission", mission)
	}
	SetLookingAroundForEnemies(plainPlayer, false)
	if timer := GetOpportunisticTimer(plainPlayer); timer != engine.NoAddress() {
		t.Errorf("GetOpportunisticTimer on a plain player = %#x, want Address_Null", timer)
	}

	if len(*touched) != 0 {
		t.Fatalf("a plain player's memory was read or written as a CTFBot's %d times", len(*touched))
	}
}

// The other half: a CTFBot still has its fields read and written.
func TestACTFBotStillHasItsFields(t *testing.T) {
	touched := installWorld(t)

	if mission := GetTFBotMission(gameBot); mission != 7 {
		t.Errorf("GetTFBotMission on a bot = %d, want the field's 7", mission)
	}
	SetLookingAroundForEnemies(gameBot, false)
	if timer := GetOpportunisticTimer(gameBot); timer == engine.NoAddress() {
		t.Error("GetOpportunisticTimer on a bot is Address_Null")
	}

	if len(*touched) != 3 {
		t.Fatalf("a bot's fields were touched %d times, want 3", len(*touched))
	}
}
