/*
Package readiness is the part of source/redbots3/nextbot_behavior.sp that
decides when a team of bots says it is ready, and when leaving the fight for a
health pack is worth it.
*/
package readiness

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// ReadyGrace is how long an unready bot gets before it is readied anyway.
//
//sp:name READY_GRACE
const ReadyGrace = 90.0

// BuildingMaxLevel is level three, where a building stops upgrading.
//
//sp:name BUILDING_MAX_LEVEL
const BuildingMaxLevel = 3

// The clock a bot's grace runs out on, per seat.
//
//sp:name m_ctReadyDeadline
var readyDeadline [slots.Count]float32

// IsBuildingFinished says the building is standing, built and at level three.
//
//sp:name IsBuildingFinished
func IsBuildingFinished(building int32) bool {
	if building == engine.InvalidEntReference() {
		return false
	}

	if engine.IsBuildingUp(building) {
		return false
	}

	return engine.EntProp(building, engine.PropSend(), "m_iUpgradeLevel") >= BuildingMaxLevel
}

// IsEngineerNestFinished is the sentry and the dispenser both done.
//
//sp:name IsEngineerNestFinished
func IsEngineerNestFinished(client int32) bool {
	return IsBuildingFinished(engine.ObjectOfType(client, engine.ObjectSentry())) &&
		IsBuildingFinished(engine.ObjectOfType(client, engine.ObjectDispenser()))
}

/*
IsDefenderPrepared says this bot has done the thing its seat exists for, before
the wave starts.

Whoever walks to the front is prepared once he is stood there: without that the
last bot to finish shopping starts the wave, and the walk to the front is
whatever fits in the time nobody is waiting for, which on Coaltown was never the
whole walk.

The engineer's teleporter counts only while nobody is being made to wait for it.
On a team of nothing but bots the between-rounds time left after the nest is
nothing at all, so requiring the teleporter meant no engineer ever finished one;
with a player on the server their shopping is already the time he needs.
*/
//
//sp:name IsDefenderPrepared
func IsDefenderPrepared(client int32) bool {
	// Credits in a pocket are worth nothing, and the whole break exists for
	// spending them.
	if engine.UseUpgrades().Bool() && !engine.ShoppedThisBreak(client) {
		return false
	}

	if engine.ShouldTakeUpPosition(client) {
		return engine.IsWaitingAtTheFront(client)
	}

	if engine.PlayerClass(client) != engine.ClassEngineer() {
		return true
	}

	if !IsEngineerNestFinished(client) {
		return false
	}

	if engine.RealPlayerCount() > 0 {
		return true
	}

	return !engine.ShouldBuildTeleporter(client)
}

/*
ReadyDefender readies a bot, and ends whatever is stopping it saying so.

A taunt holds the ready. The command goes out every frame while the flag
disagrees, so a short taunt only delays it, but a looping taunt never ends on
its own and the wave waits on a bot doing a dance. Between rounds a taunt is
worth nothing to anybody, so it loses.
*/
//
//sp:name ReadyDefender
func ReadyDefender(actor int32, state bool) {
	if state && engine.IsPlayerInCondition(actor, engine.ConditionTaunting()) {
		engine.RemoveCondition(actor, engine.ConditionTaunting())
	}

	engine.SetPlayerReady(actor, state)
}

/*
UpdateDefenderReadiness is the bots' half of the ready screen.

With a person on the team the bots are a mirror of what the people have said:
one person saying ready readies the bots, and the last person taking it back
takes theirs back too, so somebody who changes his mind gets his upgrade time
and does not have to fight six bots for it.

Past the grace a bot says it is ready rather than merely stopping being made
unready. Nothing else says it for him: an engineer whose nest will not finish
never leaves the station or moves to the front, and letting go of the ready is
not the same as pressing it, so the wave waited for him for the whole round.
*/
//
//sp:name UpdateDefenderReadiness
func UpdateDefenderReadiness(actor int32) {
	if !engine.Feature(engine.FeatureReadyWhenPrepared()) {
		return
	}

	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		readyDeadline[actor] = 0.0
		return
	}

	if engine.AnyHumanOnRed() {
		readyDeadline[actor] = 0.0

		ReadyDefender(actor, engine.AnyHumanReadyOnRed())

		return
	}

	if readyDeadline[actor] <= 0.0 {
		readyDeadline[actor] = engine.GameTime() + ReadyGrace
	}

	if engine.GameTime() > readyDeadline[actor] {
		if !engine.IsPlayerReady(actor) {
			ReadyDefender(actor, true)
		}

		return
	}

	if IsDefenderPrepared(actor) {
		return
	}

	ReadyDefender(actor, false)
}

/*
ShouldLeaveToBePatchedUp is whether leaving the fight to find a pack is worth
what the walk costs the team.

For everybody it is. For a medic it almost never is: he heals himself, so the
pack buys him what standing still would have bought him anyway, and what it
costs is the medigun for the length of the trip. Below the critical ratio he
goes anyway: a medic who dies takes the medigun with him for the rest of the
wave, which is worse than any trip.
*/
//
//sp:name ShouldLeaveToBePatchedUp
func ShouldLeaveToBePatchedUp(client int32, healthRatio float32) bool {
	if engine.PlayerClass(client) != engine.ClassMedic() {
		return true
	}

	return healthRatio < engine.HealthCriticalRatio().Float()
}
