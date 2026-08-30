/*
Package entity is the part of source/redbots3/util.sp that reads or writes one
entity: its angles, its effects, the wave stats, the revive markers.

Small reads behind names, and the enum that indexes the wave stats, which nothing
outside this file ever used.
*/
package entity

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
The wave stats record, by index.

The game keeps three of them side by side and every one is read the same way. Only
two of the five are read here, and the other three are the shipped enum: an enum
with holes in it is a set of numbers, and the numbers are what the game's record
is laid out as.
*/
//
//nolint:unused // the enum is the game's layout, not a list of things this file reads
const (
	//sp:name STATS_CREDITS_DROPPED
	statsCreditsDropped = iota
	//sp:name STATS_CREDITS_ACQUIRED
	statsCreditsAcquired
	//sp:name STATS_CREDITS_BONUS
	statsCreditsBonus
	//sp:name STATS_PLAYER_DEATHS
	statsPlayerDeaths
	//sp:name STATS_BUYBACKS
	statsBuybacks
)

// GetAbsAngles is which way the entity is facing. Its SourcePawn returns the
// array.
//
//sp:name GetAbsAngles
//sp:returns
func GetAbsAngles(entity int32) (vec [3]float32) {
	vec = engine.EntityOf(entity).AbsAngles()

	return vec
}

// RemoveEffects clears effect bits, and tells the game when that changed whether
// the entity draws.
//
//sp:name RemoveEffects
func RemoveEffects(ent int32, effects int32) {
	engine.SetEntPropSend(ent, engine.PropSend(), "m_fEffects", engine.EntProp(ent, engine.PropSend(), "m_fEffects") & ^effects)

	if effects&engine.EffectNoDraw() != 0 {
		engine.EntityOf(ent).DispatchUpdateTransmitState()
	}
}

// HasBackstabPotential says the robot can be stabbed from the front, which in
// this mode is what a stun means.
//
//sp:name HasBackstabPotential
func HasBackstabPotential(client int32) bool {
	// These are MvM-specific conditions, where stunned bots are usually allowed to be backstabbed
	if engine.PlayerTeam(client) == engine.TeamBlue() {
		if engine.IsPlayerInCondition(client, engine.ConditionRadiowave()) {
			return true
		}

		if engine.IsPlayerInCondition(client, engine.ConditionSapped()) && !engine.IsMiniBoss(client) {
			return true
		}
	}

	return false
}

// GetControlPointByID is the point entity with that index, and -1 for none.
//
//sp:name GetControlPointByID
func GetControlPointByID(pointID int32) int32 {
	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, "team_control_point")

		if ent == -1 {
			break
		}

		if engine.EntProp(ent, engine.PropData(), "m_iPointIndex") == pointID {
			return ent
		}
	}

	return -1
}

// GetNearestReviveMarker is the closest reanimator of the bot's own team within
// the distance, and -1 for none.
//
//sp:name GetNearestReviveMarker
func GetNearestReviveMarker(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)

	bestDistance := float32(999999.0)
	bestEntity := int32(-1)

	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, "entity_revive_marker")

		if ent == -1 {
			break
		}

		if engine.EntityTeamNumber(ent) != engine.GetClientTeam(client) {
			continue
		}

		distance := engine.VectorDistance(origin, engine.AbsOriginOf(ent))

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = ent
		}
	}

	return bestEntity
}

// GetBombHatchPosition is where the robots are taking it. Its SourcePawn returns
// the array.
//
//sp:name GetBombHatchPosition
//sp:returns
//sp:default useAbsOrigin false
func GetBombHatchPosition(useAbsOrigin bool) (origin [3]float32) {
	hole := engine.FindEntityByClassname(-1, "func_capturezone")

	if hole != -1 {
		if useAbsOrigin {
			origin = engine.AbsOriginOf(hole)
		} else {
			origin = engine.WorldSpaceCenter(hole)
		}
	}

	return origin
}

// GetAcquiredCreditsOfAllWaves is every credit the team has picked up this
// mission, optionally with the bonuses.
//
//sp:name GetAcquiredCreditsOfAllWaves
//sp:default withBonus true
func GetAcquiredCreditsOfAllWaves(withBonus bool) int32 {
	ent := engine.FindEntityByClassname(engine.MaxClients()+1, "tf_mann_vs_machine_stats")

	if ent == -1 {
		engine.LogError("GetAcquiredCreditsOfAllWaves: Could not find entity tf_mann_vs_machine_stats!")

		return 0
	}

	total := engine.EntPropAt(ent, engine.PropSend(), "m_runningTotalWaveStats", statsCreditsAcquired)
	total += engine.EntPropAt(ent, engine.PropSend(), "m_previousWaveStats", statsCreditsAcquired)
	total += engine.EntPropAt(ent, engine.PropSend(), "m_currentWaveStats", statsCreditsAcquired)

	if withBonus {
		total += engine.EntPropAt(ent, engine.PropSend(), "m_runningTotalWaveStats", statsCreditsBonus)
		total += engine.EntPropAt(ent, engine.PropSend(), "m_previousWaveStats", statsCreditsBonus)
		total += engine.EntPropAt(ent, engine.PropSend(), "m_currentWaveStats", statsCreditsBonus)
	}

	return total
}
