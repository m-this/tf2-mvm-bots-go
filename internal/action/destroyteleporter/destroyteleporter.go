/*
Package destroyteleporter is source/redbots3/behavior/destroyteleporter.sp.

A bot walks to an enemy teleporter and hits it. Third behaviour across, and the
first with SelectMoreDangerousThreat: the engine asks which of two things it
should worry about, and this one says the teleporter, unless a sentry is close
enough to stop it getting there.

//sp:action DefenderKillTeleporter CTFBotDestroyTeleporter
*/
package destroyteleporter

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// teleporterTarget is the teleporter each bot is going for.
//
//sp:name m_iTeleporterTarget
var teleporterTarget [Slots]int32

// OnStart aims the path and makes the bot swear at what it is about to hit.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	engine.SpeakConceptIfAllowed(actor, engine.ConceptJeers())

	return engine.Continue()
}

// Update walks to it, and gives up when it is gone or already sapped.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidEntity(teleporterTarget[actor]) || !engine.IsBaseObject(teleporterTarget[actor]) || engine.HasSapper(teleporterTarget[actor]) {
		return engine.Done("No teleporter")
	}

	myBot := engine.NextBotOf(actor)

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
		engine.RepathToTarget(actor, myBot, teleporterTarget[actor])
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// SelectMoreDangerousThreat answers which of two things to worry about.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func SelectMoreDangerousThreat(nextbot engine.Bot, entity int32, threat1 engine.Known, threat2 engine.Known) (changed engine.Outcome, knownEntity engine.Known) {
	iThreat1 := threat1.Entity()
	iThreat2 := threat2.Entity()

	me := engine.Actor()
	myWeapon := engine.ActiveWeapon(me)

	if myWeapon != -1 && (engine.WeaponID(myWeapon) == engine.WeaponFlamethrower() || engine.IsMeleeWeapon(myWeapon)) {
		// We can only get the nearest threat
		return engine.Changed(), engine.SelectCloserThreat(nextbot, threat1, threat2)
	}

	// Any sentry nearby becomes a high priority threat because it can stop us from reaching our target
	if nextbot.IsRangeLessThan(iThreat1, engine.SentryMaxRange()) && engine.IsBaseObject(iThreat1) && engine.ObjectType(iThreat1) == engine.ObjectSentry() {
		return engine.Changed(), threat1
	}

	if nextbot.IsRangeLessThan(iThreat2, engine.SentryMaxRange()) && engine.IsBaseObject(iThreat2) && engine.ObjectType(iThreat2) == engine.ObjectSentry() {
		return engine.Changed(), threat2
	}

	// Our most dangerous threat should be the teleporter
	if iThreat1 == teleporterTarget[me] && engine.IsLineOfFireClearEntity(me, engine.EyePosition(me), iThreat1) {
		return engine.Changed(), threat1
	}

	if iThreat2 == teleporterTarget[me] && engine.IsLineOfFireClearEntity(me, engine.EyePosition(me), iThreat2) {
		return engine.Changed(), threat2
	}

	// We probably can't see it right now
	return engine.Changed(), engine.NoKnownEntity()
}

// SelectTarget picks a teleporter, and only if nobody else is already on one.
//
//sp:name CTFBotDestroyTeleporter_SelectTarget
func SelectTarget(actor int32) bool {
	if engine.CountOfBotsWithNamedAction("DefenderKillTeleporter") > 0 {
		return false
	}

	teleporterTarget[actor] = engine.NearestEnemyTeleporter(actor, 5000.0)

	return teleporterTarget[actor] != -1
}
