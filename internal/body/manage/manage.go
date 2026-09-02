/*
Package manage is the seating out of source/tf2_defenderbots.sp: turning the
manager on and off, keeping RED at its size while the wave runs, and the console
command that makes one bot.

What still lives in the plugin is the three lineup walkers this calls into
(chosen, typed, preferences) and PickAllowedBotClass, which explodes a comma
list into a two dimensional char array the subset has no shape for.
*/
package manage

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// DefenderBots turns the manager on, seating the chosen lineup and starting the
// watch that keeps the team full, or off.
//
//sp:name ManageDefenderBots
//sp:default bAddBots true
func DefenderBots(bManage bool, bAddBots bool) {
	if bManage {
		if bAddBots {
			engine.AddBotsFromChosenTeamComposition()
		}

		engine.CreateTimer(1.0, TimerCheckBotImbalance, engine.Default(), engine.TimerNoMapChange()|engine.TimerRepeat())
		engine.SetBotsEnabled(true)

		engine.PrintToChatAll("%s Bots have been enabled.", engine.PluginPrefix())
	} else {
		engine.SetBotsEnabled(false)
	}
}

/*
TimerCheckBotImbalance tops RED up to its size once a second.

In the manual and ready modes bots are added before the round and watched
during it; in the auto mode they are added when the wave begins, so the watch
only runs while the round does.
*/
//
//sp:name Timer_CheckBotImbalance
//sp:public
//
//nolint:revive // unused-parameter: the timer handle is SourceMod's
func TimerCheckBotImbalance(timer engine.Timer) engine.Outcome {
	if !engine.BotsEnabled() {
		return engine.PluginStop()
	}

	switch engine.ManagerMode().Int() {
	case engine.ManagerModeManualBots(), engine.ManagerModeReadyBots():
		if engine.RoundState() != engine.RoundStateBetweenRounds() && engine.RoundState() != engine.RoundStateRunning() {
			return engine.PluginStop()
		}

		defenderCount := engine.HumanAndDefenderBotCount(engine.TeamRed())

		if defenderCount < engine.DefenderTeamSize().Int() {
			amount := engine.DefenderTeamSize().Int() - defenderCount
			AddBotsBasedOnLineupMode(amount, true)
		}
	case engine.ManagerModeAutoBots():
		if engine.RoundState() != engine.RoundStateRunning() {
			return engine.PluginStop()
		}

		defenderCount := engine.HumanAndDefenderBotCount(engine.TeamRed())

		if defenderCount < engine.DefenderTeamSize().Int() {
			amount := engine.DefenderTeamSize().Int() - defenderCount
			AddBotsBasedOnLineupMode(amount, true)
		}
	}

	return engine.PluginContinue()
}

/*
AddBotsBasedOnLineupMode fills what the named lineup left, and only that.

A three seat team and an ask for six used to be nine bots on RED: the lineup
mode filled the whole ask again on top of the named team.
*/
//
//sp:name AddBotsBasedOnLineupMode
//sp:default bAdjustTime true
func AddBotsBasedOnLineupMode(count int32, bAdjustTime bool) {
	engine.LogMessage("Fill: asked for %d, RED holds %d of %d", count, engine.HumanAndDefenderBotCount(engine.TeamRed()),
		engine.DefenderTeamSize().Int())

	count -= engine.AddBotsFromTeamComposition(count)

	if count < 1 {
		if bAdjustTime {
			engine.ExtendUpgradeTimeForNewBots()
		}

		return
	}
	engine.LogMessage("Fill: the lineup mode adds %d more", count)

	switch engine.BotLineupMode().Int() {
	case engine.LineupModeRandom():
		AddRandomDefenderBots(count)
	case engine.LineupModePreference(), engine.LineupModeChoose(), engine.LineupModePreferenceChoose():
		engine.AddBotsBasedOnPreferences(count)
	default:
		engine.ThrowError("Unhandled lineup mode %d", engine.BotLineupMode().Int())
	}

	if bAdjustTime {
		engine.ExtendUpgradeTimeForNewBots()
	}
}

/*
AddDefenderTFBot makes count bots of one class, one console command each,
because tf_bot_add cannot name several at once.

The log line says why a bot is the class it is, which nothing did: the wanted
class, what the blacklist left of it, and whether the lineup was typed into the
console or read off the map config.
*/
//
//sp:name AddDefenderTFBot
//sp:writable class
//sp:writable team
//sp:writable difficulty
//sp:default team "red"
//sp:default difficulty "expert"
//sp:default quotaManaged false
//sp:default honorBlacklist true
func AddDefenderTFBot(count int32, class engine.Text, team string, difficulty string, quotaManaged bool, honorBlacklist bool) {
	allowed := engine.CopyTextInto(class)

	if honorBlacklist {
		engine.PickAllowedBotClass(class, allowed, 512)
	}

	if !engine.StrEqualText(class, allowed) || engine.ManagerDebug().Bool() {
		typed := engine.TeamComposition().StringValue()

		engine.LogMessage("Adding %s (wanted %s), lineup from %s", allowed, class,
			engine.ChooseString(typed[0] != 0, "the convar", engine.ChooseString(engine.MapComposition()[0] != 0, "the map config", "the lineup mode")))
	}

	for i := int32(0); i < count; i++ {
		engine.ServerCommand("tf_bot_add %d %s %s %s %s %s", 1, allowed, team, difficulty, engine.ChooseString(quotaManaged, "", "noquota"), engine.TFBotIdentityName())
	}
}

// AddRandomDefenderBots seats that many bots of random classes.
//
//sp:name AddRandomDefenderBots
func AddRandomDefenderBots(amount int32) {
	engine.PrintToChatAll("%s Adding %d bot(s)...", engine.PluginPrefix(), amount)

	for i := int32(1); i <= amount; i++ {
		AddDefenderTFBot(1, engine.RawPlayerClassName(engine.Class(engine.RandomInt(1, 9))), "red", "expert", false, true)
	}
}
