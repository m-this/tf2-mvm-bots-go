/*
Package guardpoint is source/redbots3/behavior/guardpoint.sp.

A bot holds the ground around a capture point. Eighth behaviour across, and the
one that needed all four of the gaps closed: it walks a nav mesh collector, it
reads a convar, it prints with arguments, and it has two callbacks about
territory.

One deliberate difference from the shipped file. It deletes the collector before
the IsZeroVector check; the generated one deletes at each way out, so on that
path the delete happens two statements later. Neither statement in between reads
the collector, so what runs is the same, and the reason to accept the move is
that the rule holds for every path rather than the one that was written.

//sp:action DefenderGuardPoint CTFBotGuardPoint
*/
package guardpoint

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// pointDefendArea is the piece of ground each bot is holding.
//
//sp:name m_vecPointDefendArea
var pointDefendArea [slots.Count][3]float32

// OnStart finds a piece of ground near the point that the bot can actually
// reach, and gives up on the whole idea if there is none.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	point := engine.CapturableAreaTrigger(engine.PlayerEnemyTeam(actor))

	if point == -1 {
		return engine.ChangeTo(engine.DefenderAttack(), "No point found")
	}

	hAreas := engine.CollectAreasInRadius(engine.AbsOriginOf(point), 300.0)
	defer hAreas.Close()

	for i := int32(0); i < hAreas.Count(); i++ {
		area := hAreas.Get(i)

		// Don't go in spawn room
		if area.HasAttributeTF(engine.RedSpawnRoom()) || area.HasAttributeTF(engine.BlueSpawnRoom()) {
			continue
		}

		center := area.Center()

		if !engine.IsPathToVectorPossible(actor, center) {
			continue
		}

		pointDefendArea[actor] = center
		break
	}

	if engine.IsZeroVector(pointDefendArea[actor]) {
		return engine.ChangeTo(engine.DefenderAttack(), "NULL defense area")
	}

	engine.SpeakConceptIfAllowed(actor, engine.ConceptHelp())

	return engine.Continue()
}

// Update holds the ground, and gives it up for a tank or for anything worth
// shooting.
func Update(actor int32) engine.Outcome {
	switch engine.PlayerClass(actor) {
	case engine.ClassSoldier(), engine.ClassPyro(), engine.ClassDemoMan():
		if engine.AttackTankSelectTarget(actor) {
			return engine.ChangeTo(engine.AttackTank(), "Tank priority")
		}
	}

	/* Something to shoot ends this, because holding ground is what a bot does instead of fighting

	This action had no way out but a tank. Wired in as the thing a defender does when it has
	nothing to do, that would be a bot which guards the hatch once and never fights again. */
	if engine.DefenderAttackSelectTarget(actor) {
		return engine.ChangeTo(engine.DefenderAttack(), "Something to fight")
	}

	myBot := engine.NextBotOf(actor)
	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat != 0 {
		engine.EquipBestWeaponForThreat(actor, threat)
	}

	myWeapon := engine.ActiveWeapon(actor)

	// If we're close-range only, chase after them to defend the point
	if myWeapon != -1 && (engine.WeaponID(myWeapon) == engine.WeaponFlamethrower() || engine.IsMeleeWeapon(myWeapon)) {
		nearest := engine.EnemyPlayerNearestToPosition(actor, pointDefendArea[actor], 1000.0)

		if nearest != -1 {
			if engine.RepathTime(actor) <= engine.GameTime() {
				engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.5, 1.0))
				engine.RepathToTarget(actor, myBot, nearest)
			}

			engine.PathOf(actor).Update(myBot)

			return engine.Continue()
		}
	}

	// Stay near the point to defend it
	if myBot.IsRangeGreaterThanEx(pointDefendArea[actor], 200.0) {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
			engine.RepathToPos(actor, myBot, pointDefendArea[actor])
		}

		engine.PathOf(actor).Update(myBot)
	}

	return engine.Continue()
}

// OnEnd forgets the ground.
func OnEnd(actor int32) {
	pointDefendArea[actor] = engine.NullVector()
}

// OnTerritoryContested keeps holding: somebody trying to take it is the reason
// the bot is standing there.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func OnTerritoryContested(actor int32, territory int32) engine.Outcome {
	if engine.DebugActions().Bool() {
		engine.PrintToChatAll("[OnTerritoryContested] Losing CP %d", engine.ControlPointByID(territory))
	}

	// Someone tried to capture it, keep defending
	return engine.TryToSustain()
}

// OnTerritoryLost gives up.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func OnTerritoryLost(actor int32, territory int32) engine.Outcome {
	if engine.DebugActions().Bool() {
		engine.PrintToChatAll("[OnTerritoryLost] Lost CP %d!", engine.ControlPointByID(territory))
	}

	// We lost the point, give up
	return engine.TryChangeTo(engine.DefenderAttack(), engine.ResultCritical(), "Point lost")
}

// IsPossible says whether holding a point is worth doing at all.
//
//sp:name CTFBotGuardPoint_IsPossible
func IsPossible(client int32) bool {
	// There are better things for scout to do than this
	if engine.PlayerClass(client) == engine.ClassScout() {
		return false
	}

	// One of us is already watching the point
	if engine.CountOfBotsWithNamedAction("DefenderGuardPoint") > 0 {
		return false
	}

	// Nothing to defend
	if engine.CapturableAreaTrigger(engine.PlayerEnemyTeam(client)) == -1 {
		return false
	}

	// I'd rather lose the point than lose the wave!
	if engine.IsFailureImminent(client) {
		return false
	}

	return true
}

// ResetGuardPoint forgets the ground this bot was holding.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetGuardPoint(client int32) {
	pointDefendArea[client] = engine.NullVector()
}
