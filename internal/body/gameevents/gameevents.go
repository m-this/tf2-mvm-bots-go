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
