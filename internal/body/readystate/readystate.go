/*
Package readystate is the tournament ready panel's listener out of
source/tf2_defenderbots.sp.

Pressing ready is the one input the mod intercepts. In manual mode it is a floor
on how many people the mission wants; in ready-bots mode it is what summons the
bots, on the second press.
*/
package readystate

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
	ListenerTournamentPlayerReadystate decides whether a ready press goes through

Bots always pass: the mod presses ready on their behalf and blocking that would
deadlock the panel.
*/
//
//sp:name Listener_TournamentPlayerReadystate
//sp:public
//nolint:revive // unused-parameter: a command listener is handed the command and its argument count
func ListenerTournamentPlayerReadystate(client int32, command engine.Text, argc int32) engine.Outcome {
	if engine.DefenderBotFlag(client) {
		return engine.PluginContinue()
	}

	switch engine.ManagerMode().Int() {
	case engine.ManagerModeManualBots():
		if engine.ClientTeam(client) != engine.TeamRed() {
			return engine.PluginContinue()
		}

		// An admin probably added the bots.
		if engine.DefenderBotCount(engine.TeamRed()) > 0 {
			return engine.PluginContinue()
		}

		_, arg1 := engine.CmdArg(1)
		value := engine.StringToInt(arg1)

		// Zero means unready, which always passes.
		if value < 1 {
			return engine.PluginContinue()
		}

		// Allow players that are ready to unready.
		if engine.IsPlayerReady(client) {
			return engine.PluginContinue()
		}

		if engine.MinPlayers().Int() != -1 {
			difficulty := engine.MissionDifficultyNow()
			defenderTeamSize := engine.DefenderTeamSize().Int()
			minPlayers := engine.MinPlayers().Int()
			//nolint:wastedassign // the shipped function declares it before the switch and every branch writes it, which is the shape being compared
			trueMinPlayers := int32(0)

			switch difficulty {
			case engine.MissionNormal():
				// Do not go over the maximum number of RED players.
				trueMinPlayers = engine.ChooseInt(minPlayers > defenderTeamSize, defenderTeamSize, minPlayers)

				// Block ready status if there are not enough players.
				if engine.HumanAndDefenderBotCount(engine.TeamRed()) < trueMinPlayers {
					engine.PrintToChat(client, "%s More players are required.", engine.PluginPrefix())
					return engine.PluginHandled()
				}
			case engine.MissionIntermediate():
				trueMinPlayers = engine.ChooseInt(minPlayers+1 > defenderTeamSize, defenderTeamSize, minPlayers+1)

				if engine.HumanAndDefenderBotCount(engine.TeamRed()) < trueMinPlayers {
					engine.PrintToChat(client, "%s More players are required.", engine.PluginPrefix())
					return engine.PluginHandled()
				}
			case engine.MissionAdvanced():
				trueMinPlayers = engine.ChooseInt(minPlayers+2 > defenderTeamSize, defenderTeamSize, minPlayers+2)

				if engine.HumanAndDefenderBotCount(engine.TeamRed()) < trueMinPlayers {
					engine.PrintToChat(client, "%s More players are required.", engine.PluginPrefix())
					return engine.PluginHandled()
				}
			case engine.MissionExpert():
				trueMinPlayers = engine.ChooseInt(minPlayers+3 > defenderTeamSize, defenderTeamSize, minPlayers+3)

				if engine.HumanAndDefenderBotCount(engine.TeamRed()) < trueMinPlayers {
					engine.PrintToChat(client, "%s More players are required.", engine.PluginPrefix())
					return engine.PluginHandled()
				}
			case engine.MissionNightmare():
				trueMinPlayers = engine.ChooseInt(minPlayers+4 > defenderTeamSize, defenderTeamSize, minPlayers+4)

				if engine.HumanAndDefenderBotCount(engine.TeamRed()) < trueMinPlayers {
					engine.PrintToChat(client, "%s More players are required.", engine.PluginPrefix())
					return engine.PluginHandled()
				}
			default:
				engine.LogError("Listener_Readystate: Unknown difficulty returned!")
			}
		}
	case engine.ManagerModeReadyBots():
		if engine.ClientTeam(client) != engine.TeamRed() {
			return engine.PluginContinue()
		}

		if engine.DefenderBotCount(engine.TeamRed()) > 0 {
			return engine.PluginContinue()
		}

		if !engine.ShouldProcessCommand(client) {
			return engine.PluginHandled()
		}

		if engine.BotsEnabled() {
			// Bots already going, okay to pass.
			return engine.PluginContinue()
		}

		if engine.NextReadyTime() > engine.GameTime() {
			engine.PrintToChat(client, "%s You're going too fast!", engine.PluginPrefix())

			// Give more time to ready up.
			return engine.PluginHandled()
		}

		botBanTime := engine.EnableBotsCooldown(client) - engine.GameTime()

		if botBanTime > 0.0 {
			engine.ReplyToCommand(client, "%s You cannot start the bots at this time.", engine.PluginPrefix())
			engine.LogAction(client, -1, "MANAGER_MODE_READY_BOTS: %L tried to start the bots on cooldown. (%f seconds)", client, botBanTime)

			return engine.PluginHandled()
		}

		if engine.BotLineupMode().Int() == engine.LineupModeChoose() {
			if !engine.HavePlayersChosenBotTeam() {
				if engine.PlayersChoosingClasses() > 0 {
					engine.PrintToChat(client, "%s Someone is currently choosing the next team lineup.", engine.PluginPrefix())
					return engine.PluginHandled()
				}

				engine.PrintToChat(client, "%s Choose your bot team lineup first! Use command !choosebotteam or !cbt", engine.PluginPrefix())
				return engine.PluginBadLoad()
			}
		}

		if engine.LastReadyInputTime(client) <= engine.GameTime() {
			engine.SetLastReadyInputTime(client, engine.GameTime()+3.0)
			engine.PrintToChat(client, "%s Press ready again to start the bots.", engine.PluginPrefix())

			return engine.PluginHandled()
		}

		engine.ManageDefenderBots(true)
		engine.SetBotSummoner(engine.ClientUserID(client))

		return engine.PluginHandled()
	}

	return engine.PluginContinue()
}
