/*
Package spysapplayer is source/redbots3/behavior/spysapplayer.sp.

The robo sapper: a spy puts one on a player rather than a building, which stops
a giant dead. Fourth behaviour across.

//sp:action DefenderSpySapPlayer CTFBotSpySapPlayers
*/
package spysapplayer

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// groupRadius is how far a crowd counts as a crowd. The shipped code declares
// it inside SelectTarget.
const groupRadius = 800.0

// playerSapTarget is the player each spy is going for.
//
//sp:name m_iPlayerSapTarget
var playerSapTarget [slots.Count]int32

// OnStart aims the path.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	return engine.Continue()
}

// Update walks to the target and saps it once it is close enough.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidClientIndex(playerSapTarget[actor]) ||
		!engine.IsPlayerAlive(playerSapTarget[actor]) ||
		engine.PlayerTeam(playerSapTarget[actor]) != engine.PlayerEnemyTeam(actor) ||
		!engine.PlayerSappable(playerSapTarget[actor]) {
		return engine.Done("No player to sap")
	}

	mySapper := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if mySapper == -1 {
		return engine.Done("No sapper")
	}

	engine.SetPlayerActiveWeapon(actor, mySapper)

	if engine.IsStealthed(actor) || engine.IsFeignDeathReady(actor) {
		// Can't use place a sapper while cloaked, uncloak
		engine.PressAltFireButton(actor)
	} else {
		origin := engine.Origin(playerSapTarget[actor])
		myOrigin := engine.Origin(actor)

		origin = engine.SubtractVectors(origin, myOrigin)

		// If we're close enough, build a sapper on them
		if engine.VectorLength(origin) <= engine.SapperPlayerBuildOnRange() && engine.CanWeaponAttack(mySapper) {
			BuildSapperOnEntity(actor, playerSapTarget[actor], mySapper)

			return engine.Done("Sapped player")
		}
	}

	myBot := engine.NextBotOf(actor)

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 0.4))
		engine.RepathToTarget(actor, myBot, playerSapTarget[actor])
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// ShouldAttack says no: the spy is placing a sapper, not fighting.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldAttack(nextbot engine.Bot, knownEntity engine.Known) (changed engine.Outcome, result engine.Answer) {
	return engine.Changed(), engine.AnswerNo()
}

// IsHindrance avoids no one.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func IsHindrance(nextbot engine.Bot, entity int32) (changed engine.Outcome, result engine.Answer) {
	// Avoid no one
	return engine.Changed(), engine.AnswerNo()
}

// SelectTarget picks who to sap: a fast giant, then a medic with a beam out,
// then anybody in a crowd if the sapper is the robo one.
//
//sp:name CTFBotSpySapPlayers_SelectTarget
func SelectTarget(actor int32) bool {
	if !CanBuildSapper(actor) {
		return false
	}

	// Get the nearest fast giant
	playerSapTarget[actor] = engine.NearestSappablePlayer(actor, 1000.0, true, engine.ClassUnknown(), 230.0)

	// Get the nearest medic that is healing someone
	if playerSapTarget[actor] == -1 {
		playerSapTarget[actor] = engine.NearestSappablePlayerHealingSomeone(actor, 1000.0, false, engine.ClassMedic(), 0.0)
	}

	if playerSapTarget[actor] == -1 {
		secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

		if secondary != -1 && engine.WeaponID(secondary) == engine.WeaponBuilder() && engine.AttribByName(secondary, "robo sapper") != engine.NoAddress() {
			// If there's a group of enemies near us, let's put a sapper on one of them
			if engine.NearestEnemyCount(actor, groupRadius, false) >= 4 {
				playerSapTarget[actor] = engine.FarthestSappablePlayer(actor, groupRadius, false, engine.ClassUnknown(), 0.0)
			}
		}
	}

	return playerSapTarget[actor] != -1
}

// CanBuildSapper is CTFPlayer::CanBuild, only for the ammo the builder spends.
//
//sp:name CanBuildSapper
func CanBuildSapper(client int32) bool {
	// Like CTFPlayer::CanBuild, only if we have ammo of TF_AMMO_GRENADES2
	return engine.AmmoCount(client, engine.AmmoGrenades2()) > 0
}

// BuildSapperOnEntity puts one on and starts the recharge.
//
//sp:name BuildSapperOnEntity
func BuildSapperOnEntity(client int32, entity int32, weapon int32) {
	engine.SpawnSapper(client, entity, weapon)

	// CTFWeaponBuilder uses ammo index TF_AMMO_GRENADES2 for its effect bar
	engine.RemoveAmmo(client, 1, engine.AmmoGrenades2())
	StartBuilderEffectBarRegen(weapon)
}

// StartBuilderEffectBarRegen sets when the game hands the ammo back.
//
//sp:name StartBuilderEffectBarRegen
func StartBuilderEffectBarRegen(weapon int32) {
	// When recharged, game will give us ammo TF_AMMO_GRENADES2 for the sapper
	engine.SetEntPropFloat(weapon, engine.PropSend(), "m_flEffectBarRegenTime", engine.GameTime()+engine.SapperRechargeTime())
}

// ResetSpySapPlayer forgets the player's building this spy was sapping.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetSpySapPlayer(client int32) {
	playerSapTarget[client] = -1
}
