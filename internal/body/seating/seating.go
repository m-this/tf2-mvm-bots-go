/*
Package seating is the part of source/tf2_defenderbots.sp that decides when the
team may be rearranged, and the sniper hints the map config puts down.

A reseat kicks and re-adds, so it cannot run while a wave does: a bot kicked
mid-wave takes the upgrades it paid for with it. The change is held as a flag
and the break is what spends it.
*/
package seating

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
The two flags, declared here rather than in the plugin.

They were file-static in tf2_defenderbots.sp, which an included file cannot see,
and they are written from both sides: this spends them and the plugin sets them.
*/

/* reseatPending is a reseat waiting for the break, because the composition was
retyped mid-wave.

Kicking a bot in the middle of a wave loses whatever it was doing and drops its
buildings, and the replacement walks in from spawn with the bomb halfway home.
The break is where a lineup change is free. */
//
//sp:name m_bReseatPending
var reseatPending bool //nolint:unused // emitted, not read from Go

// recyclePending is a whole-team recycle waiting for the same break, asked for
// by sm_redbots_reseat.
//
//sp:name m_bRecyclePending
var recyclePending bool //nolint:unused // emitted, not read from Go

/*
	ReseatOnBreak spends a lineup change that arrived while the wave was running

Recycling wins over reseating, and clears both flags: it is the cheaper of the
two, because it reclasses the bots that are already there rather than kicking
them, and a reseat asked for on top of it would undo that.
*/
//
//sp:name Reseat_OnBreak
func ReseatOnBreak() {
	if engine.RecyclePending() {
		engine.SetRecyclePending(false)
		engine.SetReseatPending(false)
		engine.LogMessage("Reseat: recycled %d bot(s) held from mid-wave", engine.RecycleDefenderBots())
		return
	}

	if !engine.ReseatPending() {
		return
	}

	engine.SetReseatPending(false)
	engine.ReseatDefenderBots()
}

/*
	ReseatOnMapStart drops a pending change

The round it was waiting on ended with the map, and the bots it meant are gone.
*/
//
//sp:name Reseat_OnMapStart
func ReseatOnMapStart() {
	engine.SetReseatPending(false)
	engine.SetRecyclePending(false)
}

/*
	SetupSniperSpotHints puts a hint entity at every sniper spot the map names

A map with no spots in its config keeps whatever hints the mapper placed, but
they are set to team 0 so both teams may use them: an official hint is aimed at
the attackers and standing on it as a defender is usually worse than nothing.
*/
//
//sp:name SetupSniperSpotHints
func SetupSniperSpotHints() {
	if engine.SniperSpots().Length() > 0 {
		for i := int32(0); i < engine.SniperSpots().Length(); i++ {
			vec := engine.SniperSpots().GetArray(i)
			ent := engine.CreateEntityByName("func_tfbot_hint")

			if ent != -1 {
				engine.DispatchKeyValueVec(ent, "origin", vec)
				engine.DispatchKeyValue(ent, "team", "2")
				engine.DispatchKeyValue(ent, "hint", "0")
				engine.DispatchSpawn(ent)
			}
		}
	} else {
		ent := int32(-1)

		for {
			ent = engine.FindEntityByClassname(ent, "func_tfbot_hint")
			if ent == -1 {
				break
			}
			engine.DispatchKeyValue(ent, "team", "0")
		}

		engine.LogError("SetupSniperSpotHints: No hints specified by configuration, overriding other hint entities!")
	}
}

/*
	HavePlayersChosenBotTeam says the lineup is settled enough to seat bots on

A full RED is always settled, whatever anybody picked: there is no seat left to
argue about.
*/
//
//sp:name HavePlayersChosenBotTeam
func HavePlayersChosenBotTeam() bool {
	if engine.PlayersChoosingClasses() > 0 {
		return false
	}

	if engine.TeamClientCount(engine.TeamRed()) >= engine.DefenderTeamSize().Int() {
		return true
	}

	/* Strictly requiring a chosen lineup means the list only ever holds classes
	a player picked, so an empty list is nobody having chosen yet. */
	return engine.ChosenBotClasses().Length() > 0
}

// FreeChosenBotTeam drops the held lineup so it can be picked again.
//
//sp:name FreeChosenBotTeam
//sp:default bAnnounce false
func FreeChosenBotTeam(bAnnounce bool) {
	engine.ChosenBotClasses().Clear()
	engine.ChosenBotSeats().Clear()
	engine.SetBotClassesLocked(false)

	if bAnnounce {
		engine.PrintToChatAll("%s Bot team lineup can now be changed.", engine.PluginPrefix())
	}
}

/*
	FindGameConsoleVariables looks up the game's own convars once

They belong to the bot code the game ships, not to this mod, so they exist
whether or not this plugin does and there is nothing to create.
*/
//
//sp:name FindGameConsoleVariables
func FindGameConsoleVariables() {
	engine.SetBlind(engine.FindConVar("nb_blind"))
	engine.SetPathLookaheadRange(engine.FindConVar("tf_bot_path_lookahead_range"))
	engine.SetHealthCriticalRatio(engine.FindConVar("tf_bot_health_critical_ratio"))
	engine.SetHealthOkRatio(engine.FindConVar("tf_bot_health_ok_ratio"))
	engine.SetAmmoSearchRange(engine.FindConVar("tf_bot_ammo_search_range"))
	engine.SetHealthSearchFarRange(engine.FindConVar("tf_bot_health_search_far_range"))
	engine.SetHealthSearchNearRange(engine.FindConVar("tf_bot_health_search_near_range"))
}

/*
	TimerRefillDefenderTeam puts a bot back in the seat a player just left

Runs a tick after the disconnect, because the leaving player is still in the
game at the point the forward fires and would otherwise still be counted.
Nobody is left to play with if the last player leaves, so an empty defending
team is left empty rather than filled with six bots holding a hatch for no one.
*/
//
//sp:name Timer_RefillDefenderTeam
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerRefillDefenderTeam(timer engine.Timer) engine.Outcome {
	if !engine.BotsEnabled() {
		return engine.PluginStop()
	}

	if engine.RealPlayerCount() < 1 {
		return engine.PluginStop()
	}

	missing := engine.DefenderTeamSize().Int() - engine.HumanAndDefenderBotCount(engine.TeamRed())

	if missing > 0 {
		engine.AddBotsBasedOnLineupModeNow(missing, true)
	}

	return engine.PluginStop()
}

/*
	UpdateChosenBotTeamComposition decides the lineup the next fill will use

The named team is decided here, where every lineup is decided, and not where the
bots are added. The wave begins by adding this list and nothing else, so a team
named in the convar that only the top-up timer ever read was a team that never
played.
*/
//
//sp:name UpdateChosenBotTeamComposition
//sp:default caller -1
func UpdateChosenBotTeamComposition(caller int32) {
	if engine.BotClassesLocked() {
		if caller != -1 {
			engine.PrintToChat(caller, "%s Bot team lineup is locked for the next game.")
		}

		return
	}

	if engine.PlayersChoosingClasses() > 0 {
		if caller != -1 {
			engine.PrintToChat(caller, "%s Someone is currently choosing the bot team lineup.")
		}

		return
	}

	engine.ChosenBotClasses().Clear()
	engine.ChosenBotSeats().Clear()

	newBotsToAdd := engine.DefenderTeamSize().Int() - engine.HumanAndDefenderBotCount(engine.TeamRed())

	if newBotsToAdd < 1 {
		return
	}

	newBotsToAdd -= engine.CollectMissingTeamComposition(engine.ChosenBotClasses(), engine.ChosenBotSeats(), newBotsToAdd)

	// Whatever seats the named team left over are the lineup mode's to fill.
	if newBotsToAdd > 0 {
		engine.ChooseBotClassesFromLineupMode(newBotsToAdd)
	}

	if caller != -1 {
		engine.PrintToChatAll("%s %N changed the bot team lineup", engine.PluginPrefix(), caller)
	} else {
		engine.PrintToChatAll("%s Bot lineup changed", engine.PluginPrefix())
	}
}

// ChooseBotClassesFromLineupMode names count more classes for the chosen
// lineup, the way the lineup mode says to.
//
//sp:name ChooseBotClassesFromLineupMode
func ChooseBotClassesFromLineupMode(count int32) {
	switch engine.BotLineupMode().Int() {
	case engine.LineupModeRandom():
		for i := int32(1); i <= count; i++ {
			engine.ChosenBotClasses().PushStringText(engine.RawPlayerClassName(engine.RandomClassBetween(engine.ClassScout(), engine.ClassEngineer())))
		}
	case engine.LineupModePreference(), engine.LineupModePreferenceChoose():
		adtClassPref := engine.NewListSized(engine.ClassNameMax())

		engine.CollectPlayerBotClassPreferences(adtClassPref)

		if adtClassPref.Length() > 0 {
			for i := int32(1); i <= count; i++ {
				class := adtClassPref.GetString(engine.RandomInt(0, adtClassPref.Length()-1))

				engine.ChosenBotClasses().PushStringText(class)
			}
		} else {
			for i := int32(1); i <= count; i++ {
				engine.ChosenBotClasses().PushStringText(engine.RawPlayerClassName(engine.RandomClassBetween(engine.ClassScout(), engine.ClassEngineer())))
			}
		}

		adtClassPref.Close()
	default:
		engine.ThrowError("Unknown lineup mode %d", engine.BotLineupMode().Int())
	}
}
