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

/*
	MakePlayerDance is the send-off on the final wave

The taunt itself is not written yet, so this is the shape and the aliveness test
and nothing else. It stays as the shipped file has it: mvm-z83.41 says a port
does not carry a fix, and filling this in is a fix.

*/
//
//sp:name MakePlayerDance
func MakePlayerDance(client int32) {
	if engine.IsPlayerAlive(client) {
		_ = client
	}
}

/*
	MakeRoomForHumanPlayer frees a defender seat for somebody who just connected

The bots are not kicked between waves any more, because a kicked bot is a bot
that paid for its upgrades and left them behind. That leaves nothing to open a
seat: the game caps the defending team, the bots fill it, and a player who joins
the server after the mission started is told the team is full for the rest of it.

Only when RED is already full: below the size there is a seat going spare and
nobody has to leave. A dead bot goes first, since kicking one costs the team
nothing it still had.

*/
//
//sp:name MakeRoomForHumanPlayer
func MakeRoomForHumanPlayer(client int32) {
	if engine.HumanAndDefenderBotCount(engine.TeamRed()) < engine.DefenderTeamSize().Int() {
		return
	}

	victim := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client || !engine.IsClientInGame(i) || !engine.DefenderBotFlag(i) {
			continue
		}

		if engine.ClientTeam(i) != engine.TeamRed() {
			continue
		}

		if !engine.IsPlayerAlive(i) {
			victim = i
			break
		}

		if victim == -1 {
			victim = i
		}
	}

	if victim == -1 {
		return
	}

	engine.KickClient(victim, "BotManager3: Making room for a player")
}

// RemoveAllDefenderBots empties RED of ours.
//
//sp:name RemoveAllDefenderBots
//sp:writable reason
//sp:default reason ""
//sp:default bDanceInstead false
func RemoveAllDefenderBots(reason engine.Text, bDanceInstead bool) {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.DefenderBotFlag(i) {
			// The final wave dances instead of leaving.
			if bDanceInstead {
				engine.MakePlayerDance(i)
				continue
			}

			engine.KickClientText(i, reason)
		}
	}
}
