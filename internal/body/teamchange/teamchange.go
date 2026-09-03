/*
Package teamchange is what source/tf2_defenderbots.sp does when somebody moves
between teams while the round is waiting to start.

Mann vs Machine begins the wave when every member of the defending team has
pressed ready, so a team that is entirely ready leaves no room for another bot
to walk in. One member is unreadied to hold the door open, a human before a bot,
and a bot that was unreadied readies itself again a fifth of a second later.
*/
package teamchange

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// TimerReadyPlayer readies the bot that was unreadied to make room.
//
//sp:name Timer_ReadyPlayer
//sp:public
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerReadyPlayer(timer engine.Timer, data int32) {
	if !engine.IsClientInGame(data) {
		return
	}

	engine.SetPlayerReady(data, true)
}

// HandleTeamPlayerCountChanged is the whole of it.
//
//sp:name HandleTeamPlayerCountChanged
//sp:default iWhoChanging -1
func HandleTeamPlayerCountChanged(team engine.Team, iWhoChanging int32) {
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return
	}

	if engine.ManagerMode().Int() == engine.ManagerModeManualBots() {
		if iWhoChanging > 0 && iWhoChanging == engine.ClientOfUserID(engine.BotSummoner()) && engine.VoteInProgress() {
			// He started the bot vote then changed teams, cancel it.
			engine.CancelVote()
		}
	}

	switch engine.BotLineupMode().Int() {
	case engine.LineupModeChoose(), engine.LineupModePreferenceChoose():
		// Allow the classes to be picked again, but do not clear the list.
		engine.SetBotClassesLocked(false)
		engine.PrintToChatTeam(int32(team), "%s You can repick your bot team lineup.", engine.PluginPrefix())
	}

	if !engine.BotsEnabled() {
		return
	}

	if iWhoChanging > 0 && engine.ClientOfUserID(engine.BotSummoner()) == iWhoChanging {
		// The summoner changed teams, so RED may repick its bots.
		engine.SetAllowBotRedo(true)
		engine.PrintToChatTeam(int32(team), "%s Use command !redobots to repick your bot team lineup.", engine.PluginPrefix())
	}

	iWhoToUnready := int32(-1)
	iReadyCount := int32(0)
	iMemberCount := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		// Whoever is changing teams does not count to the team count.
		if i == iWhoChanging {
			continue
		}

		if !engine.IsClientInGame(i) {
			continue
		}

		if engine.ClientTeam(i) != team {
			continue
		}

		if engine.IsPlayerReady(i) {
			if iWhoToUnready != -1 {
				if engine.DefenderBotFlag(iWhoToUnready) {
					// Always prefer to unready human players first.
					if !engine.DefenderBotFlag(i) {
						iWhoToUnready = i
					}
				}
			} else {
				iWhoToUnready = i
			}

			iReadyCount++
		}

		iMemberCount++
	}

	// Are all remaining members of the team ready?
	if iReadyCount == iMemberCount {
		// Unready one so the wave cannot start and another bot may enter.
		engine.SetPlayerReady(iWhoToUnready, false)

		if engine.DefenderBotFlag(iWhoToUnready) {
			engine.CreateTimerVoid(0.2, TimerReadyPlayer, iWhoToUnready, engine.TimerNoMapChange())
		}
	}
}
