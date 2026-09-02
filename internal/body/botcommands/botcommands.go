/*
Package botcommands is the console commands out of source/tf2_defenderbots.sp
that only show something or hand off.

The ones that decide anything stay with what they decide about; these are the
handles a player or an admin reaches the rest of the mod by.
*/
package botcommands

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// CommandBotPreferences opens the preference menu.
//
//sp:name Command_BotPreferences
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandBotPreferences(client int32, args int32) engine.Outcome {
	engine.DisplayMenu(engine.BotPreferenceMenuOf(), client, engine.MenuTimeForever())
	return engine.PluginHandled()
}

// CommandShowBotChances shows each class's share of the draw.
//
//sp:name Command_ShowBotChances
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandShowBotChances(client int32, args int32) engine.Outcome {
	engine.ShowCurrentBotClassChances(client)
	return engine.PluginHandled()
}

// CommandShowNewBotTeamComposition shows the lineup, or says there is none.
//
//sp:name Command_ShowNewBotTeamComposition
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandShowNewBotTeamComposition(client int32, args int32) engine.Outcome {
	if !engine.DisplayPanelBotTeamComposition(client) {
		engine.ReplyToCommand(client, "%s There is no bot lineup currently active.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	engine.ReplyToCommand(client, "Use command !rerollbotclasses to reshuffle the bot class lineup.")

	return engine.PluginHandled()
}

/*
CommandAddBots adds a number of bots, or opens the menu to pick them one at a
time.

The break is not extended for a manual add: an admin adding bots mid-break knows
what they are doing, and stretching the round timer under them is a surprise.
*/
//
//sp:name Command_AddBots
//sp:public
func CommandAddBots(client int32, args int32) engine.Outcome {
	if args > 0 {
		_, arg1 := engine.CmdArg(1)
		amount := engine.StringToInt(arg1)
		engine.AddBotsBasedOnLineupModeNow(amount, false)

		return engine.PluginHandled()
	}

	engine.DisplayMenuAddDefenderBots(client)
	return engine.PluginHandled()
}

// CommandRemoveAllBots kicks the team, and turns the manager off first when
// asked to.
//
//sp:name Command_RemoveAllBots
//sp:public
func CommandRemoveAllBots(client int32, args int32) engine.Outcome {
	if args > 0 {
		_, arg1 := engine.CmdArg(1)

		if engine.StringToInt(arg1) == 1 {
			engine.ManageDefenderBotsOn(false)
		}
	}

	engine.RemoveAllDefenderBotsFor("Admin request")
	engine.ShowActivity2(client, "[SM] ", "Purged all bots.")

	return engine.PluginHandled()
}

// CommandStopManagingBots turns the manager off and leaves the team standing.
//
//sp:name Command_StopManagingBots
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandStopManagingBots(client int32, args int32) engine.Outcome {
	engine.ManageDefenderBotsOn(false)
	engine.ReplyToCommand(client, "Stopped manaing bots.")

	return engine.PluginHandled()
}
