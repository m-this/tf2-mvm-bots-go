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
