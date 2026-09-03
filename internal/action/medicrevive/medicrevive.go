/*
Package medicrevive is source/redbots3/behavior/medicrevive.sp.

A medic walks to a reanimator and holds the beam on it until the dead defender
is back. Sixth behaviour across, and the first with OnInjured: something hitting
the medic mid-revive is when the uber is worth popping.

//sp:action DefenderMedicRevive CTFBotMedicRevive
*/
package medicrevive

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

//sp:name MEDIC_REVIVE_RANGE
const reviveRange = 600.0

/*
askInterval is how long an answer about reachability is held for.

The reachability test is a full NavAreaBuildPath and this runs on the tactical
monitor's frame, twice: once from the game's heal action and once from the mod's.
A wave leaves revive markers lying about wherever a defender died, so a medic in
a fight had one in range most of the time and paid for a nav mesh search on every
frame of it to be told what it was told last frame.

That is the third call of this shape found in one night, after the health and
ammo search and the nest scoring. The pattern is worth naming: a nav mesh
question inside something that reads like a cheap predicate, on a path that runs
every frame.

Half a second, which is a medic walking about a hundred and fifty units. A
marker does not appear and expire inside that.
*/
//
//sp:name MEDIC_REVIVE_ASK_INTERVAL
const askInterval = 0.5

// The memo the interval above protects.
var (
	//sp:name m_ctReviveAsk
	reviveAsk [slots.Count]float32
	//sp:name m_bRevivePossible
	revivePossible [slots.Count]bool
)

// OnStart aims the path.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	return engine.Continue()
}

// Update holds the beam on the marker, and pulls the primary out on the way
// there so the medic is not defenceless.
func Update(actor int32) engine.Outcome {
	secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if secondary == -1 {
		return engine.Done("No medigun!")
	}

	marker := engine.NearestReviveMarker(actor, reviveRange)

	if marker == -1 {
		return engine.Done("No reanimator!")
	}

	markerPos := engine.WorldSpaceCenter(marker)
	myBot := engine.NextBotOf(actor)

	if myBot.IsRangeLessThanEx(markerPos, engine.WeaponMedigunRange()) {
		healTarget := engine.EntPropEnt(secondary, engine.PropSend(), "m_hHealingTarget")

		// The empty branch is the shipped file's, and it is the point: a
		// medic already beaming something else stops pressing fire, and
		// writing that as a negated condition would read as a different
		// decision to anybody diffing the two.
		//
		//nolint:revive // empty-block: kept as shipped, see above
		if healTarget != -1 && healTarget != marker {
			// We're healing something that's not the revive marker, stop holding the attack button
		} else {
			engine.SetPlayerActiveWeapon(actor, secondary)
			engine.SnapViewToPosition(actor, markerPos)
			engine.PressFireButton(actor)
		}

		// Do not path if we are healing our target
		if healTarget == marker {
			return engine.Continue()
		}
	} else {
		// Fend off from enemies
		primary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotPrimary())

		if primary != -1 {
			engine.SetPlayerActiveWeapon(actor, primary)
		}
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.9, 1.2))
		engine.RepathToPos(actor, myBot, markerPos)
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnInjured pops the uber when something hits the medic mid-revive.
func OnInjured(actor int32, takedamageinfo int32) engine.Outcome {
	info := engine.DamageOf(takedamageinfo)

	if info.Amount() > 0.0 {
		weapon := engine.ActiveWeapon(actor)

		// Someone hit me while I'm trying to revive someone, let's pop uber now if possible
		if weapon != -1 && engine.WeaponID(weapon) == engine.WeaponMedigun() {
			engine.PressAltFireButton(actor)
		}
	}

	return engine.Continue()
}

// IsPossible answers whether there is anybody to revive, held for askInterval.
//
//sp:name CTFBotMedicRevive_IsPossible
func IsPossible(client int32) bool {
	if reviveAsk[client] > engine.GameTime() {
		return revivePossible[client]
	}

	reviveAsk[client] = engine.GameTime() + askInterval

	revivePossible[client] = false

	marker := engine.NearestReviveMarker(client, reviveRange)

	if marker == -1 {
		return false
	}

	if !engine.IsPathToVectorPossible(client, engine.AbsOriginOf(marker)) {
		return false
	}

	revivePossible[client] = true

	return true
}
