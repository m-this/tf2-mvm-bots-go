/*
Package attackforuber is source/redbots3/behavior/attackforuber.sp.

A medic with a half-full charge and nobody shooting at him walks up to something
and hits it, because some melee weapons fill the uber on hit. Fifth behaviour
across, and the first with a vector per client as its state.

//sp:action DefenderAttackUber CTFBotAttackUber
*/
package attackforuber

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// The two the shipped file defines at the top.
const (
	//sp:name MEDIC_ATTACK_UBER_LOW_HEALTH
	lowHealth = 100
	//sp:name MEDIC_ATTACK_UBER_SEEK_RANGE
	seekRange = 500.0
)

// startArea is where the medic was standing when this began, so he can be told
// to come back rather than chase somebody across the map.
//
//sp:name m_vecStartArea
var startArea [Slots][3]float32

// OnStart aims the path and remembers where the medic is.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	startArea[actor] = engine.Origin(actor)

	return engine.Continue()
}

// Update finds something to hit and hits it, and gives up on any of the six
// reasons this stops being a good idea.
func Update(actor int32) engine.Outcome {
	if engine.ClientHealth(actor) < lowHealth && !engine.IsInvulnerable(actor) {
		return engine.Done("Low health")
	}

	secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if secondary == -1 || engine.WeaponID(secondary) != engine.WeaponMedigun() {
		return engine.Done("No medigun")
	}

	myChargeLevel := engine.EntPropFloatOf(secondary, engine.PropSend(), "m_flChargeLevel")

	if myChargeLevel >= 1.0 {
		return engine.Done("Full uber")
	}

	melee := engine.PlayerWeaponSlot(actor, engine.WeaponSlotMelee())

	if melee == -1 {
		return engine.Done("No melee")
	}

	engine.SetPlayerActiveWeapon(actor, melee)

	myBot := engine.NextBotOf(actor)

	// Let's not stray too far from the patient
	if myBot.IsRangeGreaterThanEx(startArea[actor], seekRange) {
		return engine.Done("Too far from home")
	}

	target := engine.EnemyNearestToMe(actor, seekRange, false, true, false, engine.ClassUnknown())

	if target == -1 {
		return engine.Done("Nobody near me")
	}

	if myBot.IsRangeLessThan(target, engine.MeleeAttackRange()) {
		engine.SnapViewToPosition(actor, engine.WorldSpaceCenter(target))

		if myChargeLevel < 0.5 && myBot.IsRangeLessThan(target, 100.0) && !engine.IsPlayerMoving(target) {
			// Attempt to do a taunt kill on them for the full uber
			if !engine.IsTaunting(actor) {
				engine.PressAltFireButton(actor)
			} else {
				return engine.Continue()
			}
		} else {
			engine.PressFireButton(actor)
		}
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 1.0))
		engine.RepathToTarget(actor, myBot, target)
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnEnd forgets where he started.
func OnEnd(actor int32) {
	startArea[actor] = engine.NullVector()
}

// IsPossible is the eight questions asked before this is worth starting.
//
//sp:name CTFBotAttackUber_IsPossible
func IsPossible(client int32, medigun int32) bool {
	isUbered := engine.IsInvulnerable(client)

	// Health is too low
	if !isUbered && engine.ClientHealth(client) < lowHealth {
		return false
	}

	// I should be healing someone first
	if engine.EntPropEnt(medigun, engine.PropSend(), "m_hHealingTarget") == -1 {
		return false
	}

	// It's already full
	if engine.EntPropFloatOf(medigun, engine.PropSend(), "m_flChargeLevel") >= 1.0 {
		return false
	}

	// We are already using ubercharge
	if engine.EntProp(medigun, engine.PropSend(), "m_bChargeRelease") == 1 {
		return false
	}

	melee := engine.PlayerWeaponSlot(client, engine.WeaponSlotMelee())

	if melee == -1 {
		return false
	}

	if !engine.CanWeaponAddUberOnHit(melee) {
		return false
	}

	// Too dangerous
	if !isUbered && engine.NearestEnemyCount(client, 1000.0, false) > 2 {
		return false
	}

	if engine.EnemyNearestToMe(client, seekRange, false, true, false, engine.ClassUnknown()) == -1 {
		return false
	}

	return true
}
