/*
Package markgiant is source/redbots3/behavior/markgiant.sp.

The Fan O'War puts a mark on whoever it hits, and a marked giant takes a good
deal more from everybody else. A scout carrying one picks a giant and goes for
it. Eleventh behaviour across.

//sp:action DefenderMarkGiant CTFBotMarkGiant
*/
package markgiant

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// fanOWar is the item definition index of the Fan O'War.
const fanOWar = 355

//sp:name m_iTarget
var target [Slots]int32

//sp:name m_flNextMarkTime
var nextMarkTime [Slots]float32

// OnStart picks a giant at random out of the ones worth marking.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	weapon := MarkForDeathWeapon(actor)

	if weapon == engine.InvalidEntReference() {
		return engine.Done("Don't have a mark-for-death weapon")
	}

	potentialVictims := engine.NewList()
	defer potentialVictims.Close()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor {
			continue
		}

		if !engine.IsClientInGame(i) {
			continue
		}

		if PlayerMarkable(actor, i) {
			potentialVictims.Push(i)
		}
	}

	if potentialVictims.Length() == 0 {
		target[actor] = -1
		return engine.Done("No eligible mark victims")
	}

	target[actor] = potentialVictims.Get(engine.RandomInt(0, potentialVictims.Length()-1))

	engine.EquipWeaponSlot(actor, engine.WeaponSlotMelee())

	return engine.Continue()
}

// Update walks at the giant, and takes the game's own idea of what the bot has
// noticed away for a moment so it looks at the right one.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidClientIndex(target[actor]) || !engine.IsPlayerAlive(target[actor]) {
		target[actor] = -1
		return engine.Done("Mark target is no longer valid")
	}

	if !PlayerMarkable(actor, target[actor]) {
		target[actor] = -1
		return engine.Done("Mark target is no longer markable")
	}

	myOrigin := engine.Origin(actor)
	targetOrigin := engine.Origin(target[actor])

	distToTarget := engine.VectorDistance(myOrigin, targetOrigin)

	myBot := engine.NextBotOf(actor)

	if distToTarget < 512.0 {
		// TODO: aim directly on target instead of doing this dumb shit
		myVision := myBot.Vision()

		if myVision.KnownCount(engine.TeamBlue()) > 1 || myVision.GetKnown(target[actor]) == engine.NoKnownEntity() {
			myVision.ForgetAllKnownEntities()
			myVision.AddKnownEntity(target[actor])
		}
	}

	// TODO: stop pathing once we reached the desired attack range
	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
		engine.RepathToTarget(actor, myBot, target[actor])
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnEnd puts the mark on a cooldown so a scout does not spend the wave doing it.
func OnEnd(actor int32) {
	nextMarkTime[actor] = engine.GameTime() + 30.0
	target[actor] = -1
}

// MarkForDeathWeapon is the Fan O'War, if the bot has one.
//
//sp:name GetMarkForDeathWeapon
func MarkForDeathWeapon(player int32) int32 {
	for i := int32(0); i < 8; i++ {
		weapon := engine.PlayerWeaponSlot(player, i)

		if !engine.IsValidEntity(weapon) {
			continue
		}

		itemDefinitionIndex := engine.EntProp(weapon, engine.PropSend(), "m_iItemDefinitionIndex")

		if itemDefinitionIndex == fanOWar {
			return weapon
		}
	}

	return engine.InvalidEntReference()
}

// PlayerMarkable is the seven questions asked of a possible victim.
//
//sp:name IsPlayerMarkable
func PlayerMarkable(bot int32, victim int32) bool {
	if nextMarkTime[bot] < engine.GameTime() {
		return false
	}

	if !engine.IsClientInGame(victim) {
		return false
	}

	if !engine.IsPlayerAlive(victim) {
		return false
	}

	if engine.EntityTeamNumber(bot) == engine.EntityTeamNumber(victim) {
		return false
	}

	if !engine.IsMiniBoss(victim) {
		return false
	}

	if engine.IsSentryBusterRobot(victim) {
		return false
	}

	if engine.IsPlayerInCondition(victim, engine.ConditionMarkedForDeath()) {
		return false
	}

	if engine.IsInvulnerable(victim) {
		return false
	}

	return true
}

// IsPossible says whether there is a giant worth marking.
//
//sp:name CTFBotMarkGiant_IsPossible
func IsPossible(actor int32) bool {
	if MarkForDeathWeapon(actor) == engine.InvalidEntReference() {
		return false
	}

	victimExists := false

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor {
			continue
		}

		if !engine.IsClientConnected(i) {
			continue
		}

		if PlayerMarkable(actor, i) {
			victimExists = true
		}
	}

	return victimExists
}

// ResetMarkGiant forgets the giant this bot marked, and when it may mark again.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetMarkGiant(client int32) {
	target[client] = -1
	nextMarkTime[client] = 0.0
}
