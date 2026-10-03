package upgrade

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
A shopping trip can end after its bot has left: the seat refill kicks bots and
the end of a mission punts them all, and the action's OnEnd still runs.
FakeClientCommandKeyValues throws on a client that is not connected, which is
apw-bjf's "Client 24 is not connected" out of KV_MvM_UpgradesDone. Every call
but the one asking whether he is in game is left without an answer, so any
other call fails the test by name.
*/
func TestATripEndingAfterTheBotLeftSendsNothing(_ *testing.T) {
	restore := engine.Install(engine.Calls{
		IsClientInGame: func(int32) bool { return false },
	})
	defer restore()
	restorePlayers := engine.InstallPlayers(engine.PlayerCalls{})
	defer restorePlayers()
	restorePrefs := engine.InstallPrefs(engine.PrefCalls{})
	defer restorePrefs()
	restoreStations := engine.InstallStations(engine.StationCalls{})
	defer restoreStations()

	OnEnd(24)
}

// The other half: a bot still in the game closes the session with the count.
func TestATripEndingInGameClosesTheSession(t *testing.T) {
	const actor = 7

	restore := engine.Install(engine.Calls{
		IsClientInGame: func(int32) bool { return true },
		IsPlayerAlive:  func(int32) bool { return false },
		PlayerClass:    func(int32) engine.Class { return engine.ClassSoldier() },
	})
	defer restore()

	var sent []string
	var counted int32
	target := int32(-1)
	restorePlayers := engine.InstallPlayers(engine.PlayerCalls{
		NewKeyValues:   func(name string) engine.KeyValues { sent = append(sent, name); return 1 },
		FakeCommandKV:  func(client int32, _ engine.KeyValues) { target = client },
		CloseKeyValues: func(engine.KeyValues) {},
	})
	defer restorePlayers()
	restorePrefs := engine.InstallPrefs(engine.PrefCalls{
		SetNum: func(_ engine.KeyValues, key string, value int32) {
			if key == "num_upgrades" {
				counted = value
			}
		},
	})
	defer restorePrefs()

	purchasedUpgrades[actor] = 3
	defer func() { purchasedUpgrades[actor] = 0 }()

	OnEnd(actor)

	if len(sent) != 1 || sent[0] != "MvM_UpgradesDone" || target != actor {
		t.Fatalf("OnEnd sent %v to client %d, want MvM_UpgradesDone to client %d", sent, target, actor)
	}
	if counted != 3 {
		t.Errorf("num_upgrades = %d, want 3", counted)
	}
}
