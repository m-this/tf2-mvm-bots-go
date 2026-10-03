package lifecycle

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
A bot whose behaviour has been built and not started has no actor bound to it,
and the game's own answer to GetPrimaryKnownThreat reads that actor: apw-4ei is
the server reading 0x2668 off a null CTFBot from this hook (mvm-9q4). Nothing
about the nextbot is installed here, so a run-command that asks the bot's vision
anything fails the test by name.
*/
func TestARunCommandBeforeTheBehaviourStartsAsksTheBotNothing(t *testing.T) {
	const client = 5

	started := false
	asked := false
	restore := installRunCmd(t, func(c int32) bool {
		if c != client {
			t.Errorf("MainActionStarted(%d), want %d", c, client)
		}
		asked = true
		return started
	})
	defer restore()

	OnPlayerRunCmd(client, 0, 0, [3]float32{}, [3]float32{}, 0, 0, 0, 0, 0, [2]int32{})

	if !asked {
		t.Fatal("the run-command never asked whether the behaviour had started")
	}
}

// The other half: once it has started, the bot's vision is asked as before.
func TestARunCommandAfterTheBehaviourStartsAsksTheVision(t *testing.T) {
	const client = 5

	restore := installRunCmd(t, func(int32) bool { return true })
	defer restore()

	threatAsked := false
	restoreBots := engine.InstallBots(engine.BotCalls{
		NextBotOf: func(int32) engine.Bot { return 1 },
		Vision:    func(engine.Bot) engine.Vision { return 2 },
		PrimaryKnownThreat: func(engine.Vision, bool) engine.Known {
			threatAsked = true
			return engine.NoKnownEntity()
		},
	})
	defer restoreBots()

	func() {
		defer func() { _ = recover() }() // what follows the threat is not this test's business
		OnPlayerRunCmd(client, 0, 0, [3]float32{}, [3]float32{}, 0, 0, 0, 0, 0, [2]int32{})
	}()

	if !threatAsked {
		t.Fatal("a started bot was never asked for its threat")
	}
}

// installRunCmd answers everything a living defender's run-command asks before
// it reaches the bot, mid-wave, holding nothing.
func installRunCmd(t *testing.T, started func(int32) bool) func() {
	t.Helper()

	restores := []func(){
		engine.Install(engine.Calls{
			IsPlayerAlive: func(int32) bool { return true },
			ActiveWeapon:  func(int32) int32 { return -1 },
			WeaponID:      func(int32) engine.Weapon { return -1 },
		}),
		engine.InstallFronts(engine.FrontCalls{
			DefenderBotFlag:          func(int32) bool { return true },
			LookupEntityActionByName: func(int32, string) int32 { return 0 },
		}),
		engine.InstallPutInServers(engine.PutInServerCalls{
			Press:   func(int32) int32 { return 0 },
			Release: func(int32) int32 { return 0 },
		}),
		engine.InstallRunCmds(engine.RunCmdCalls{
			WatchDefenderSpawnExit: func(int32) {},
			PluginBotSimulateFrame: func(int32) {},
			MainActionStarted:      started,
			MonitorKnownEntities:   func(int32, engine.Vision) {},
		}),
		engine.InstallStations(engine.StationCalls{
			RoundState: func() int32 { return 4 },
		}),
		engine.InstallBusters(engine.BusterCalls{
			IsInUpgradeZone: func(int32) bool { return false },
		}),
		engine.InstallBots(engine.BotCalls{}),
	}
	return func() {
		for i := len(restores) - 1; i >= 0; i-- {
			restores[i]()
		}
	}
}
