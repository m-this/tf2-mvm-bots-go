/*
Package finders is the part of source/redbots3/util.sp that answers "is there one
of those near me": a medic's patient, a friendly dispenser, a sentry buster, a
medic beam already on you.

Each is a loop over clients or entities with one distance test, and each was an
extern the ported behaviours reached for.
*/
package finders

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How far a medigun beam reaches.
//
//sp:name MEDIGUN_HEAL_RANGE
const medigunHealRange = 450.0

// How far from a dispenser still counts as standing at it, and how hurt a bot has
// to be before it wants one.
const (
	// Named in the default of FindFriendlyDispenserNear, which SourcePawn
	// reads and Go has no way to spell in a signature.
	//
	//sp:name DISPENSER_GUARD_RANGE
	//
	//nolint:unused // the emitted default is the use
	dispenserGuardRange = 600.0
	//sp:name DISPENSER_GUARD_HEALTH_RATIO
	dispenserGuardHealthRatio = 0.8
)

// MedicHasPatient says the medic is healing somebody, or is close enough to
// somebody that he is about to be.
//
//sp:name MedicHasPatient
func MedicHasPatient(client int32, medigun int32) bool {
	if engine.EntPropEnt(medigun, engine.PropSend(), "m_hHealingTarget") != -1 {
		return true
	}

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client || !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.GetClientTeam(i) != engine.GetClientTeam(client) {
			continue
		}

		if engine.VectorDistance(engine.WorldSpaceCenter(client), engine.WorldSpaceCenter(i)) < medigunHealRange {
			return true
		}
	}

	return false
}

// FindFriendlyDispenserNear is the nearest finished dispenser of the bot's own
// team, and -1 for none in range.
//
//sp:name FindFriendlyDispenserNear
//sp:default maxRange DISPENSER_GUARD_RANGE
func FindFriendlyDispenserNear(client int32, origin [3]float32, maxRange float32) int32 {
	bestDistance := maxRange
	best := int32(-1)

	dispenser := int32(-1)

	for {
		dispenser = engine.FindEntityByClassname(dispenser, "obj_dispenser")

		if dispenser == -1 {
			break
		}

		if engine.EntProp(dispenser, engine.PropSend(), "m_bPlacing") != 0 || engine.EntProp(dispenser, engine.PropSend(), "m_bBuilding") != 0 {
			continue
		}

		if engine.EntityTeamNumber(dispenser) != engine.GetClientTeam(client) {
			continue
		}

		distance := engine.VectorDistance(engine.AbsOriginOf(dispenser), origin)

		if distance < bestDistance {
			bestDistance = distance
			best = dispenser
		}
	}

	return best
}

// WantsDispenser says the bot is hurt or low on ammo, which are the two reasons
// to stand at one.
//
//sp:name WantsDispenser
func WantsDispenser(client int32) bool {
	if float32(engine.ClientHealth(client)) < float32(engine.EntityMaxHealth(client))*dispenserGuardHealthRatio {
		return true
	}

	return engine.IsAmmoLow(client)
}

// FindSentryBusterNear is the nearest buster of the other team that has left its
// spawn, and -1 for none in range.
//
//sp:name FindSentryBusterNear
func FindSentryBusterNear(origin [3]float32, enemyTeam engine.Team, maxRange float32) int32 {
	bestDistance := maxRange
	best := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.PlayerTeam(i) != enemyTeam {
			continue
		}

		if !engine.IsSentryBusterRobot(i) {
			continue
		}

		if engine.IsPointInRespawnRoom(engine.WorldSpaceCenter(i)) {
			continue
		}

		distance := engine.VectorDistance(engine.WorldSpaceCenter(i), origin)

		if distance < bestDistance {
			bestDistance = distance
			best = i
		}
	}

	return best
}

// IsHealedByMedic says a person rather than a dispenser is healing this bot.
//
//sp:name IsHealedByMedic
func IsHealedByMedic(client int32) bool {
	for i := int32(0); i < engine.NumHealers(client); i++ {
		healerIndex := engine.PlayerHealer(client, i)

		// Not a player.
		if !engine.IsPlayer(healerIndex) {
			continue
		}

		return true
	}

	return false
}
