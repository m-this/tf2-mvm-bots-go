/*
Package commands is the console commands out of source/tf2_defenderbots.sp.

Each answers wherever it was run from rather than in chat, because rcon has no
client and printing to one is printing to nobody.
*/
package commands

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
UpgradesMax is the count past which the game's answer is not believable.

Declared here rather than in the plugin: the generated file is included long
before the line the define used to sit on, and a define has to precede its
reader.
*/
//
//sp:name DUMP_UPGRADES_MAX
const UpgradesMax = 1024

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

	//nolint:gocritic // singleCaseSwitch: the shipped function is a switch, and an if cannot be compared against it
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

// CommandJoinBluePlayWithBots puts the caller on BLU and fills RED with bots,
// which is the one-player-against-the-mod game.
//
//sp:name Command_JoinBluePlayWithBots
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandJoinBluePlayWithBots(client int32, args int32) engine.Outcome {
	if engine.ManagerMode().Int() < engine.ManagerModeManualBots() {
		engine.ReplyToCommand(client, "%s Currently not allowed.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.BotsEnabled() {
		engine.ReplyToCommand(client, "%s Bots are already enabled for this round.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ClientTeam(client) != engine.TeamBlue() {
		engine.ReplyToCommand(client, "%s Your team is not allowed to use this.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.HumanAndDefenderBotCount(engine.TeamRed()) > 0 {
		engine.ReplyToCommand(client, "%s You cannot use this with players on RED team.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	engine.AddRandomDefenderBots(engine.DefenderTeamSize().Int())
	engine.SetBotsEnabled(true)
	engine.PrintToChatAll("%s You will play a game with bots.", engine.PluginPrefix())

	return engine.PluginHandled()
}

/*
	CommandChooseBotClasses opens the lineup menu for a solo player

Only between waves and only while solo, so the current team count is always one
and the menu asks for the rest of the seats.
*/
//
//sp:name Command_ChooseBotClasses
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandChooseBotClasses(client int32, args int32) engine.Outcome {
	if engine.BotsEnabled() {
		engine.ReplyToCommand(client, "%s Bots are already enabled.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.BotLineupMode().Int() != engine.LineupModeChoose() {
		engine.ReplyToCommand(client, "%s Not allowed in the current manager lineup mode.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ClientTeam(client) != engine.TeamRed() {
		engine.ReplyToCommand(client, "%s Your team is not allowed to use this.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.BotClassesLocked() {
		engine.ReplyToCommand(client, "%s Someone has already chosen the lineup for the next game.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ChoosingBotClasses(client) {
		engine.ReplyToCommand(client, "%s You are already choosing the next team lineup.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.PlayersChoosingClasses() > 0 {
		engine.ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		engine.ReplyToCommand(client, "%s This can only be used between waves.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	redTeamCount := engine.HumanAndDefenderBotCount(engine.TeamRed())
	defenderTeamSize := engine.DefenderTeamSize().Int()

	if redTeamCount >= defenderTeamSize {
		engine.ReplyToCommand(client, "%s You are not solo.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	engine.ShowDefenderBotTeamSetupMenu(client, 0, true, defenderTeamSize-redTeamCount)
	engine.PrintToChatAll("%N is choosing the current bot team lineup.", client)

	return engine.PluginHandled()
}

// CommandRedoBotTeamLineup throws the current bots away and picks again.
//
//sp:name Command_RedoBotTeamLineup
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandRedoBotTeamLineup(client int32, args int32) engine.Outcome {
	if !engine.BotsEnabled() {
		engine.ReplyToCommand(client, "%s The bots aren't here, dummy.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if !engine.AllowBotRedo() {
		engine.ReplyToCommand(client, "%s This is currently not allowed.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ClientTeam(client) != engine.TeamRed() {
		engine.ReplyToCommand(client, "%s Your team is not allowed to use this.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ChoosingBotClasses(client) {
		engine.ReplyToCommand(client, "%s You are already choosing the next team lineup.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.PlayersChoosingClasses() > 0 {
		engine.ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	switch engine.BotLineupMode().Int() {
	case engine.LineupModeRandom():
		engine.SetBotsEnabled(false)
		engine.RemoveAllDefenderBotsFor("DB redo bots")
		engine.SetBotClassesLocked(false)
		engine.UpdateChosenBotTeamComposition()
	case engine.LineupModePreference():
		engine.SetBotsEnabled(false)
		engine.RemoveAllDefenderBotsFor("DB redo bots")
		engine.SetBotClassesLocked(false)
		engine.UpdateChosenBotTeamComposition()
	case engine.LineupModeChoose():
		engine.SetBotsEnabled(false)
		engine.RemoveAllDefenderBotsFor("DB redo bots")
		engine.FreeChosenBotTeamAnnouncing(false)
		CommandChooseBotClasses(client, 0)
	case engine.LineupModePreferenceChoose():
		engine.SetBotsEnabled(false)
		engine.RemoveAllDefenderBotsFor("DB redo bots")
		engine.SetBotClassesLocked(false)
		engine.UpdateChosenBotTeamComposition()
	}

	// Solo players are always allowed to repick their bot lineup.
	engine.SetAllowBotRedo(engine.TeamHumanClientCount(engine.TeamRed()) == 1)

	engine.PrintToChatAll("%s %N has decided to repick the bot team lineup.", engine.PluginPrefix(), client)
	engine.LogAction(client, -1, "%L triggered defender bot redo", client)

	return engine.PluginHandled()
}

/*
	CommandVotebots asks the server to vote the bots in

Manual mode only, and only from RED between waves. The lineup has to be settled
first when the mode says a player picks it, because a vote that passes on an
empty lineup would seat nobody.
*/
//
//sp:name Command_Votebots
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandVotebots(client int32, args int32) engine.Outcome {
	if engine.BotsEnabled() {
		engine.ReplyToCommand(client, "%s Bots are already enabled for this round.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.ManagerMode().Int() != engine.ManagerModeManualBots() {
		engine.ReplyToCommand(client, "%s This is only allowed in MANAGER_MODE_MANUAL_BOTS.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.NextReadyTime() > engine.GameTime() {
		engine.ReplyToCommand(client, "%s You're going too fast!", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.IsServerFull() {
		engine.ReplyToCommand(client, "%s Server is at max capacity.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		engine.ReplyToCommand(client, "%s This cannot be used at this time.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.VoteInProgress() {
		engine.ReplyToCommand(client, "%s A vote is already in progress.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.BotLineupMode().Int() == engine.LineupModeChoose() {
		if !engine.HavePlayersChosenBotTeam() {
			if engine.ChoosingBotClasses(client) {
				engine.ReplyToCommand(client, "%s You are already choosing the next team lineup.", engine.PluginPrefix())
				return engine.PluginHandled()
			}

			if engine.PlayersChoosingClasses() > 0 {
				engine.ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", engine.PluginPrefix())
				return engine.PluginHandled()
			}

			engine.ReplyToCommand(client, "%s Choose your bot team lineup first! Use command !choosebotteam or !cbt", engine.PluginPrefix())
			return engine.PluginBadLoad()
		}
	}

	switch engine.ClientTeam(client) {
	case engine.TeamRed():
		botBanTime := engine.EnableBotsCooldown(client) - engine.GameTime()

		if botBanTime > 0.0 {
			engine.ReplyToCommand(client, "%s You cannot start the bots at this time.", engine.PluginPrefix())
			engine.LogAction(client, -1, "MANAGER_MODE_MANUAL_BOTS: %L tried to start the bots on cooldown. (%f seconds)", client, botBanTime)

			return engine.PluginHandled()
		}

		if engine.HumanAndDefenderBotCount(engine.TeamRed()) < engine.DefenderTeamSize().Int() {
			engine.StartBotVote(client)
			return engine.PluginHandled()
		}

		engine.ReplyToCommand(client, "%s RED team is full.", engine.PluginPrefix())
		return engine.PluginHandled()
	default:
		engine.ReplyToCommand(client, "%s You cannot use this command on this team.", engine.PluginPrefix())
		return engine.PluginHandled()
	}
}

/*
	CommandRequestExtraBot adds one bot over the team size

The named class is checked before anything is added, so a typo says so rather
than quietly seating a random one.
*/
//
//sp:name Command_RequestExtraBot
//sp:public
func CommandRequestExtraBot(client int32, args int32) engine.Outcome {
	if !engine.BotsEnabled() {
		engine.ReplyToCommand(client, "%s Bots aren't enabled.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.AddingBotTime() > engine.GameTime() {
		return engine.PluginHandled()
	}

	if engine.ClientTeam(client) != engine.TeamRed() {
		engine.ReplyToCommand(client, "%s Your team is not allowed to use this.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	if engine.IsServerFull() {
		engine.ReplyToCommand(client, "%s It is currently not possible to add any more.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	defenderLimit := engine.DefenderTeamSize().Int() + engine.ExtraBots().Int()

	if engine.HumanAndDefenderBotCount(engine.TeamRed()) >= defenderLimit {
		engine.ReplyToCommand(client, "%s You already have an additional bot.", engine.PluginPrefix())
		return engine.PluginHandled()
	}

	engine.SetAddingBotTime(engine.GameTime() + 0.1)

	if args > 0 {
		_, arg1 := engine.CmdArg(1)

		if engine.CompareTextCased(arg1, "random", false) == 0 {
			engine.AddRandomDefenderBots(1)
			return engine.PluginHandled()
		}

		class := engine.ClassIndexFromString(arg1)

		if class == engine.ClassUnknown() {
			engine.ReplyToCommand(client, "%s Invalid class specified: %s.", engine.PluginPrefix(), arg1)
			return engine.PluginHandled()
		}

		engine.AddDefenderTFBotClass(1, arg1)
		engine.PrintToChatAll("%s %N requested an additional \"%s\" bot.", engine.PluginPrefix(), client, arg1)

		return engine.PluginHandled()
	}

	engine.AddBotsBasedOnLineupModeCount(1)
	engine.PrintToChatAll("%s %N requested an additional bot.", engine.PluginPrefix(), client)

	return engine.PluginHandled()
}

/*
	CommandDumpUpgrades prints the game's own upgrade list, by the index it uses

The count is the manager's raw one rather than UpgradeCount's: this command
exists to say when the game gives an answer that is not believable, and
UpgradeCount is the place that hides it.
*/
//
//sp:name Command_DumpUpgrades
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandDumpUpgrades(client int32, args int32) engine.Outcome {
	if !engine.IsUpgradeManagerUp() {
		engine.ReplyToCommand(client, "[SM] The upgrade manager is not up yet. Load an MvM map first.")
		return engine.PluginHandled()
	}

	count := engine.UpgradeCountRaw()

	if count < 1 || count > UpgradesMax {
		engine.ReplyToCommand(client, "[SM] The manager says it holds %d upgrades, which is not believable.", count)
		return engine.PluginHandled()
	}

	engine.ReplyToCommand(client, "[SM] %d upgrades, by the index the game uses:", count)
	engine.LogMessage("sm_dump_upgrades: %d upgrades", count)

	for i := int32(0); i < count; i++ {
		upgrade := engine.UpgradeByIndex(i)

		var attribute engine.Text

		if upgrade != engine.NoAddress() {
			attribute = engine.UpgradeAttribute(upgrade)
		}

		engine.ReplyToCommand(client, "%d %s", i, engine.ChooseText(attribute[0] == 0, "(none)", attribute))
		engine.LogMessage("%d %s", i, engine.ChooseText(attribute[0] == 0, "(none)", attribute))
	}

	return engine.PluginHandled()
}

// Targets is MAXPLAYERS, the width of the list a pattern resolves into.
const Targets = 101

/*
	CommandViewBotUpgrades prints what one or more players are carrying

The pattern is SourceMod's, so @all and #userid work, and the answer is filtered
to the living: an upgrade on a corpse is not what anybody is asking about.
*/
//
//sp:name Command_ViewBotUpgrades
//sp:public
func CommandViewBotUpgrades(client int32, args int32) engine.Outcome {
	if args < 1 {
		engine.ReplyToCommand(client, "[SM] Usage: sm_view_bot_upgrades <#userid|name> <slot>")
		return engine.PluginHandled()
	}

	_, arg := engine.CmdArg(1)

	var targetName engine.Text
	var targetList [Targets]int32
	var tnIsML bool

	targetCount := engine.ProcessTargetString(arg, client, targetList, engine.MaxTargets(), engine.CommandFilterAlive(), targetName, engine.TextSize(), tnIsML)

	if targetCount <= 0 {
		engine.ReplyToTargetError(client, targetCount)
		return engine.PluginHandled()
	}

	slot := int32(-1)

	if args >= 2 {
		_, arg2 := engine.CmdArg(2)

		slot = engine.StringToInt(arg2)
	}

	for i := int32(0); i < targetCount; i++ {
		engine.ShowPlayerUpgrades(client, targetList[i], slot)
	}

	return engine.PluginHandled()
}
