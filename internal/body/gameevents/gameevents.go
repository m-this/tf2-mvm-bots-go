/*
Package gameevents is the part of source/redbots3/events.sp that answers a game
event and needs nothing else to do it.

The handlers that reach half the plugin are still in the file. These four are the
ones whose whole job is on the event.
*/
package gameevents

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
OpenTheBreak gives everybody a fresh shopping trip.

Cleared when a break opens rather than when a wave begins, which is when a break
ends. The other way round, every bot that lived through a wave was still marked
as having shopped for the whole of the next break, and there is nothing else to
offer an engineer or a medic between rounds: they stood where the wave left them
until it started again. A bot that died shopped normally, because a spawn clears
the same flag, which is what made it look intermittent.
*/
//
//sp:name OpenTheBreak
func OpenTheBreak() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		engine.SetShoppedThisBreak(i, false)
	}
}

// EventRevivePlayerNotify notes that somebody attempted a revive on this player.
//
//sp:name Event_RevivePlayerNotify
//sp:public
//nolint:revive // unused-parameter: the name and the broadcast flag are SourceMod's, and this reads neither
func EventRevivePlayerNotify(event engine.Event, name string, dontBroadcast bool) {
	client := event.EventInt("entindex")

	engine.SetBeingRevived(client, true)
}

// EventMvmMissionUpdate blocks the event while a defender spy is dying, because
// TFBot spies fire it on death.
//
//sp:name Event_MvmMissionUpdate
//sp:public
//nolint:revive // unused-parameter: the event and its name are SourceMod's, and this reads neither
func EventMvmMissionUpdate(event engine.Event, name string, dontBroadcast bool) engine.Outcome {
	if engine.SpyKilled() {
		return engine.PluginHandled()
	}

	return engine.PluginContinue()
}

// EventTeamplayRoundStart forgets the last wave's spies, and reads the map again
// when the map itself was reset.
//
//sp:name Event_TeamplayRoundStart
//sp:public
//nolint:revive // unused-parameter: the name and the broadcast flag are SourceMod's, and this reads neither
func EventTeamplayRoundStart(event engine.Event, name string, dontBroadcast bool) {
	// A new wave has its own Spies, and the last wave's paranoia is not
	// evidence about this one.
	engine.ResetSpyIntel()

	// Was the map reset?
	if event.EventBool("full_reset") {
		engine.SetupSniperSpotHints()
		engine.NestRelocationResetAll()
	}
}

/*
EventMvmWaveBegin is everything that has to happen on the frame a wave starts,
and one thing that deliberately does not.

Resetting an intention throws away a bot's behaviour and has it rebuilt on its
next update, and rebuilding runs the OnStart of whatever it picks. Several of
those are not cheap. Doing it for six bots inside the wave_begin frame puts all
of it on the one frame of a mission that is already the most expensive: every
robot spawns there and starts pathing at the same moment. Three runs of an A/B
died on exactly that frame, so the resets are a queue drained a bot a tick.
*/
//
//sp:name Event_MvmWaveBegin
//sp:public
//
//nolint:revive // unused-parameter: the event and its name are SourceMod's, and this reads neither
func EventMvmWaveBegin(event engine.Event, name string, dontBroadcast bool) {
	// Nothing unless a debug convar is set, which is never on a real server.
	engine.DebugFaultsOnWaveStart()
	engine.DebugFaultsOnWaveStartEmpty()

	/* Published here rather than only on a timer after the map loads

	server.cfg runs at its own pace and a late-loaded plugin misses it
	entirely, so a list published once on map start can be the defaults rather
	than what the server was asked for. A wave beginning is after everything,
	every time. */
	engine.PublishActiveFeatures()
	engine.ThreatPortAuditReport()

	// Whatever the queue has left is about a bomb that is about to move.
	engine.NestRelocationStopEvaluating()

	// A new wave is a new chance at a spot that refused him last time.
	engine.TeleporterForgetGivingUp()
	engine.DisposableForgetGivingUp()

	// One a tick, because the frame this runs on is the one the server dies on.
	engine.QueueBehaviourReset()

	// A hat the game refused is an edict nobody will ever free, and there is
	// one per refusal.
	engine.RemoveOrphanedWearables()

	if engine.ManagerMode().Int() == engine.ManagerModeAutoBots() {
		engine.ManageDefenderBots(true)
	}

	// At this point the bots should already be here, so clear up the lineup
	// that was used.
	engine.FreeChosenBotTeam()
}

// The wave failures since the counter was last cleared, which tells a mission
// restart apart from a wave the players simply lost.
//
//sp:name m_iWaveFailCounterTick
//nolint:unused // Event_MvmWaveFailed counts it up, and that one is still in the plugin
var waveFailCounterTick int32

/*
TimerWaveFailure hands the bots' upgrades back after a wave they lost.

Not necessary in itself: the point is that the population manager forgets what
they bought, so they go and buy again in their upgrade behaviour. It is really
for the bots that failed a wave and were not kicked.
*/
//
//sp:name Timer_WaveFailure
//sp:public
//
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerWaveFailure(timer engine.Timer) engine.Outcome {
	waveFailCounterTick = 0

	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return engine.PluginStop()
	}

	// Don't refund if we wanna keep them.
	if engine.KeepBotUpgrades().Bool() {
		return engine.PluginStop()
	}

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.DefenderBotFlag(i) {
			if engine.HasUpgraded(i) {
				engine.SetHasBoughtUpgrades(i, false)
				engine.GrantOrRemoveAllUpgrades(i, true, true)
				engine.SetHasUpgraded(i, false)
			}
		}
	}

	return engine.PluginStop()
}

// TimerUpdateChosenBotTeamComposition works out the next lineup, unless the
// players are picking it themselves.
//
//sp:name Timer_UpdateChosenBotTeamComposition
//sp:public
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerUpdateChosenBotTeamComposition(timer engine.Timer) engine.Outcome {
	// These modes use their own way of composing a bot team.
	if engine.BotLineupMode().Int() == engine.LineupModeChoose() {
		return engine.PluginStop()
	}

	engine.UpdateChosenBotTeamComposition()

	return engine.PluginStop()
}

/*
EventMvmWaveFailed is the break that opens when the players lose a wave.

The same wave comes back down the same route, so there is nothing new to say
about the nests.
*/
//
//sp:name Event_MvmWaveFailed
//sp:public
//
//nolint:revive // unused-parameter: the event and its name are SourceMod's, and this reads neither
func EventMvmWaveFailed(event engine.Event, name string, dontBroadcast bool) {
	OpenTheBreak()

	// A lineup retyped mid-wave was held until now.
	engine.ReseatOnBreak()

	waveFailCounterTick++

	engine.NestRelocationResetAll()

	if engine.KickBots().Bool() {
		engine.RemoveAllDefenderBots("BotManager3: Wave failed!")
		engine.ManageDefenderBots(false)
		engine.CreateTimer(0.1, TimerUpdateChosenBotTeamComposition, engine.Default(), engine.TimerNoMapChange())
		engine.PrintToChatAll("%s Use command !viewbotlineup to view the next bot team composition", engine.PluginPrefix())
	}

	if engine.ManagerMode().Int() == engine.ManagerModeReadyBots() {
		// Global cooldown before players can ready up again.
		engine.SetNextReadyTime(engine.GameTime() + engine.ReadyCooldown().Float())

		if waveFailCounterTick > 3 {
			// Mission restarted or changed, don't have a cooldown here.
			engine.SetNextReadyTime(0.0)
		}
	}

	if engine.BotLineupMode().Int() == engine.LineupModeChoose() {
		// In case the mission changed, let players pick the bot team.
		engine.FreeChosenBotTeam()
	}

	engine.CreateTimer(0.1, TimerWaveFailure, engine.Default(), engine.TimerNoMapChange())
}

/*
InitGameEventHooks is every game event the mod listens for.

The mission update is hooked Pre because it is the one the mod changes rather
than only reads.
*/
//
//sp:name InitGameEventHooks
func InitGameEventHooks() {
	engine.HookEvent("player_spawn", EventPlayerSpawn)
	engine.HookEvent("mvm_wave_failed", EventMvmWaveFailed)
	engine.HookEvent("mvm_wave_complete", EventMvmWaveComplete)
	engine.HookEvent("revive_player_notify", EventRevivePlayerNotify)
	engine.HookEvent("mvm_begin_wave", EventMvmWaveBegin)
	engine.HookEvent("player_team", EventPlayerTeam)
	engine.HookEventPre("mvm_mission_update", EventMvmMissionUpdate)
	engine.HookEvent("teamplay_round_start", EventTeamplayRoundStart)
	engine.HookEvent("player_death", EventPlayerDeath)
}

/*
ListenerVoiceMenu hears a player press the medic call.

The first entry of the first menu is "MEDIC!", and it is the only one worth
reading: everything else on the wheel is a bot saying something to nobody.
*/
//
//sp:name Listener_VoiceMenu
//
//nolint:revive // unused-parameter: the command name is SourceMod's, and this listener is registered for one
func ListenerVoiceMenu(client int32, command string, argc int32) engine.Outcome {
	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) {
		return engine.PluginContinue()
	}

	if argc < 2 || engine.IsTFBotPlayer(client) {
		return engine.PluginContinue()
	}

	_, menu := engine.CmdArg(1)
	_, entry := engine.CmdArg(2)

	if engine.StringToInt(menu) == 0 && engine.StringToInt(entry) == 0 {
		engine.NoteMedicCall(client)
	}

	return engine.PluginContinue()
}

/*
EventPlayerSpawn is where a bot is recognised, dressed and given its shopping
budget.

The identification runs a fifth of a second later because the popfile is still
building the robot when this fires, and a bend applied to a half-built robot is
a bend the game overwrites.
*/
//
//sp:name Event_PlayerSpawn
//
//nolint:revive // unused-parameter: the event name and the broadcast flag are SourceMod's
func EventPlayerSpawn(event engine.Event, name string, dontBroadcast bool) {
	client := engine.ClientOfUserID(event.EventInt("userid"))

	if engine.ClientTeam(client) == engine.TeamRed() && engine.IsTFBotPlayer(client) {
		engine.CreateTimerWith(0.2, TimerPlayerSpawn, client, engine.TimerNoMapChange())
	}

	// The popfile is still building this robot, so the bend waits a frame.
	if engine.ClientTeam(client) == engine.TeamBlue() && engine.IsFakeClient(client) {
		engine.BluAssistOnRobotSpawn(client)
	}

	if engine.DefenderBotFlag(client) {
		engine.GiveBotCosmeticsSoon(client)

		engine.SetBeingRevived(client, false)
		engine.SetBuyUpgradesNumber(client, engine.ChooseInt(engine.CanBuyUpgradesNow(client), engine.RandomInt(1, 100), 0))

		if engine.ManagerDebug().Bool() {
			engine.PrintToChatAll("[Event_PlayerSpawn] g_iBuyUpgradesNumber[%d] = %d", client, engine.BuyUpgradesNumber(client))
		}
	}
}

/*
EventMvmWaveComplete opens the break.

The nest relocation is asked before anything sends the engineers off to shop:
the shopping trip is what tears their buildings down, and it needs this answer
to know whether it should.
*/
//
//sp:name Event_MvmWaveComplete
//
//nolint:revive // unused-parameter: the event, its name and the broadcast flag are SourceMod's
func EventMvmWaveComplete(event engine.Event, name string, dontBroadcast bool) {
	OpenTheBreak()

	// A lineup retyped mid-wave was held until now.
	engine.ReseatOnBreak()

	engine.EngineerNestRelocationOnWaveComplete()

	bRequestCredits := engine.RequestCredits().Bool()

	if engine.KickBots().Bool() {
		engine.RemoveAllDefenderBotsWhen("BotManager3: Wave complete!", engine.IsFinalWave())
		engine.ManageDefenderBotsOn(false)
		engine.CreateTimer(0.1, TimerUpdateChosenBotTeamComposition, engine.Default(), engine.TimerNoMapChange())
		engine.PrintToChatAll("%s Use command !viewbotlineup to view the next bot team composition", engine.PluginPrefix())
	}

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.DefenderBotFlag(i) {
			// Wave complete, rethink what we should do.
			engine.ClearSniperStall(i)
			engine.ResetIntentionInterface(i)

			if bRequestCredits {
				engine.FakeClientCommandText(i, "sm_requestcredits")
			}
		}
	}
}

/*
EventPlayerTeam keeps the lineup in step with who is actually playing.

Only for people: a bot joining RED is the mod's own doing and does not change
what the lineup should be.
*/
//
//sp:name Event_PlayerTeam
//
//nolint:revive // unused-parameter: the event name and the broadcast flag are SourceMod's
func EventPlayerTeam(event engine.Event, name string, dontBroadcast bool) {
	client := engine.ClientOfUserID(event.EventInt("userid"))
	team := engine.Team(event.EventInt("team"))
	oldTeam := engine.Team(event.EventInt("oldteam"))
	isDisconnect := event.EventBool("disconnect")

	/* A managed defender belongs on RED for its whole connection.

	If the game moves one elsewhere, RED is one seat short but the misplaced
	client still occupies a server slot. On a full server the fill timer then
	requests a replacement it cannot create. Remove only that bot; the normal
	imbalance path recreates the empty RED seat.

	An intentional kick also reports a team change. Leave disconnects alone so
	this does not turn every removal into a second kick. */
	if engine.IsFakeClient(client) {
		if !isDisconnect && engine.DefenderBotFlag(client) && oldTeam == engine.TeamRed() && team != engine.TeamRed() {
			engine.ClearBuildingsBeforeKick(client)
			engine.KickClient(client, "BotManager3: restoring the RED lineup")
		}

		return
	}

	/* When changing teams, update the bot team composition for a RED
	player who disconnected, a player who joined RED, and a player who
	left RED. */
	if (isDisconnect && oldTeam == engine.TeamRed()) || (!isDisconnect && (team == engine.TeamRed() || oldTeam == engine.TeamRed())) {
		engine.CreateTimer(0.1, TimerUpdateChosenBotTeamComposition, engine.Default(), engine.TimerNoMapChange())

		if oldTeam == engine.TeamRed() {
			engine.HandleTeamPlayerCountChanged(engine.TeamRed(), client)
		}
	}

	/* Switching from BLUE to RED bars the player from starting the bots
		for a while

	A player who cannot get the team they want by asking used to get it by
	joining BLUE, starting the bots and coming back. The cooldown grows
	each time rather than resetting, so doing it repeatedly costs more
	each go. */
	if !isDisconnect && team == engine.TeamRed() && oldTeam == engine.TeamBlue() && !engine.CheckCommandAccess(client, engine.NullString(), engine.AdmFlagGeneric()) {
		if engine.EnableBotsCooldown(client) <= engine.GameTime() {
			engine.SetEnableBotsCooldown(client, engine.GameTime()+30.0)
		} else {
			engine.SetEnableBotsCooldown(client, engine.EnableBotsCooldown(client)+10.0)
		}
	}
}

/*
EventPlayerDeath puts the team on alert when a robot spy stabs somebody.

A robot's Spy, and not the team's own. The rule was "a Spy killed somebody on
another team", which a defending Spy satisfies every time he stabs a robot, so
the team's own Spy put everybody on alert and the bots then frisked each other
and him. Reported from play by somebody trying to play Spy: "your teammates keep
trying to call you out as an enemy spy".

Measured before the fix: a lineup with a friendly Spy in it spent 5.4 per cent
of its samples spy checking, and two lineups without one spent none at all
across eight thousand samples.
*/
//
//sp:name Event_PlayerDeath
//
//nolint:revive // unused-parameter: the event name and the broadcast flag are SourceMod's
func EventPlayerDeath(event engine.Event, name string, dontBroadcast bool) {
	attacker := engine.ClientOfUserID(event.EventInt("attacker"))
	victim := engine.ClientOfUserID(event.EventInt("userid"))

	if !engine.IsValidClientIndex(attacker) || !engine.IsValidClientIndex(victim) || attacker == victim {
		return
	}

	if engine.PlayerClass(attacker) != engine.ClassSpy() {
		return
	}

	if engine.ClientTeam(attacker) != engine.TeamBlue() {
		return
	}

	origin := engine.Origin(victim)

	engine.NoteSpySighting(origin)
}

/*
TimerPlayerSpawn is where a bot the server made becomes one of ours.

A fifth of a second after the spawn, because the popfile is still building the
robot when the event fires. The identity name is what says a bot is ours: the
server makes it and the mod recognises it afterwards.

The credits are set by hand because CTFGameRules::GetTeamAssignmentOverride
ignores bot players, so a bot joining RED gets none of what the wave has paid.
The third term is Archipelago's, and zero on a server without that plugin: the
game's own record never saw a Cash Bundle, so without it a bot that rejoins or
changes class comes back with every bundle it was paid missing.
*/
//
//sp:name Timer_PlayerSpawn
//sp:public
//
//nolint:revive,gocritic // unused-parameter, elseif: the timer handle is SourceMod's, and the nesting is the shipped shape
func TimerPlayerSpawn(timer engine.Timer, data int32) engine.Outcome {
	if !engine.IsClientInGame(data) || !engine.IsTFBotPlayer(data) || engine.ClientTeam(data) != engine.TeamRed() {
		return engine.PluginStop()
	}

	if engine.DefenderBotFlag(data) {
		// Mainly for wave failures, try to request credits again.
		if engine.RequestCredits().Bool() && engine.RoundState() == engine.RoundStateBetweenRounds() {
			engine.FakeClientCommandText(data, "sm_requestcredits")
		}

		if engine.ManagerDebug().Bool() {
			engine.PrintToChatAll("[Timer_PlayerSpawn] %N's currency: %d", data, engine.Currency(data))
		}

		// We already made this one into our bot, so do nothing.
		return engine.PluginStop()
	}

	_, clientName := engine.ClientName(data)

	// Identify if the bot is ours.
	if engine.StrContains(clientName, engine.TFBotIdentityName(), true) != -1 {
		engine.SetDefenderBotFlag(data, true)
		engine.SetHasBoughtUpgrades(data, false)

		// The spawn that identified this bot ran before the flag above was
		// set, so its cosmetics were skipped.
		engine.GiveBotCosmeticsSoon(data)

		if engine.UseCustomLoadouts().Bool() {
			// Custom weapons are not given unless the player respawns again.
			engine.RespawnPlayer(data)
		} else {
			// Without custom loadouts the sniper only ever uses a rifle, and
			// the custom path runs its own check for that.
			if engine.PlayerClass(data) == engine.ClassSniper() {
				engine.SetMission(data, engine.MissionSniper())
			}
		}

		// Let medic bots use their shields.
		engine.AddBotAttribute(data, engine.BotProjectileShield())

		engine.MarkNeedsNamePurge(data)

		engine.SetCurrencyWithBundles(data, engine.StartingCurrency(engine.PopulationManager())+engine.AcquiredCreditsOfAllWaves())

		// Field of view of 90. The vision FOV updates in
		// CTFBotMainAction::Update from m_iFOV.
		engine.SetFakeClientConVar(data, "fov_desired", "90")

		engine.HookTouchPost(data)

		engine.HookDefenderBot(data)

		if engine.RequestCredits().Bool() {
			engine.FakeClientCommandText(data, "sm_requestcredits")
		}

		if engine.IsValidAttributeName("cannot be sapped") {
			engine.SetAttribByName(data, "cannot be sapped", 1.0)
		}

		engine.SetRandomNameOnBotFor(data)
	}

	return engine.PluginStop()
}
