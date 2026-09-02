/*
Package commands is the console commands out of source/tf2_defenderbots.sp.

Each answers wherever it was run from rather than in chat, because rcon has no
client and printing to one is printing to nobody.
*/
package commands

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// CommandRerollNewBotTeamComposition picks the lineup again and shows it.
//
//sp:name Command_RerollNewBotTeamComposition
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandRerollNewBotTeamComposition(client int32, args int32) engine.Outcome {
	if engine.ClientTeam(client) != engine.TeamRed() {
		engine.ReplyToCommand(client, "%s Your team is not allowed to use this.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	switch engine.BotLineupMode().Int() {
	case engine.LineupModeChoose():
		engine.ReplyToCommand(client, "%s This cannot be used with the current lineup mode.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	engine.UpdateChosenBotTeamCompositionFor(client)
	engine.DisplayPanelBotTeamComposition(client)

	return engine.PluginHandled()
}

/*
	CommandForcePlayerPreference makes one player's preferences stand for everybody's

Only @me is wired up. The admin form, forcing somebody else's, is not written.
*/
//
//sp:name Command_ForcePlayerPreference
//sp:public
func CommandForcePlayerPreference(client int32, args int32) engine.Outcome {
	if args < 1 {
		engine.ReplyToCommand(client, "[SM] Usage: sm_db_use_pref_of_player <#userid|name>")
		return engine.PluginHandled()
	}

	_, arg := engine.CmdArg(1)

	// We only want one target at a time here.
	if engine.CompareTextTo(arg, "@me") == 0 {
		engine.SetPlayerForcedPref(client)
		return engine.PluginHandled()
	}

	return engine.PluginHandled()
}

// CommandDumpCredits prints what everybody on RED is holding, and what the
// mission has paid out so far.
//
//sp:name Command_DumpCredits
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandDumpCredits(client int32, args int32) engine.Outcome {
	earned := engine.StartingCurrency(engine.PopulationManager()) + engine.AcquiredCreditsOfAllWaves()

	engine.ReplyToCommand(client, "[SM] starting plus acquired is %d, before anything Archipelago paid", earned)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || engine.ClientTeam(i) != engine.TeamRed() {
			continue
		}

		_, name := engine.ClientName(i)

		engine.ReplyToCommand(client, "[SM] %-20s %-14s %6d credits%s", name, engine.RawPlayerClassName(engine.PlayerClass(i)),
			engine.Currency(i), engine.Choose(engine.IsDefenderBot(i), "", " (human)"))
	}

	return engine.PluginHandled()
}

/*
	CommandReseatBots rebuilds the team from the loadout file

A recycle asked for mid-wave is held until the break: kicking a bot in the
middle of one loses whatever it was doing and drops its buildings.
*/
//
//sp:name Command_ReseatBots
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandReseatBots(client int32, args int32) engine.Outcome {
	engine.ConfigLoadServerLoadout()

	if engine.RoundState() == engine.RoundStateRunning() {
		engine.SetRecyclePending(true)
		engine.ReplyToCommand(client, "%s The new team takes effect when this wave ends.", engine.PluginPrefix())
		engine.PrintToChatAll("%s The new team takes effect when this wave ends.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	kicked := engine.RecycleDefenderBots()
	engine.LogMessage("Reseat: the team was retyped, recycled %d bot(s)", kicked)
	engine.ReplyToCommand(client, "%s Rebuilding %d bot(s) from the new team.", engine.PluginPrefix(), kicked)

	if kicked > 0 {
		engine.PrintToChatAll("%s Rebuilding %d bot(s) from the new team...", engine.PluginPrefix(), kicked)
	}

	return engine.PluginHandled()
}
