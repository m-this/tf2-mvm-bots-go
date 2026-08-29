/*
Package campbomb is source/redbots3/behavior/campbomb.sp.

A bot stands on the ground around a dropped bomb and shoots whatever comes for
it. Seventh behaviour across, and the first that hands the engine a different
behaviour: a tank inbound outranks a bomb on the floor.

//sp:action DefenderCampBomb CTFBotCampBomb
*/
package campbomb

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
The three the shipped file defines at the top.

Only the guard radius is read here. BOMB_HATCH_RANGE_CRITICAL is read twice by
nextbot_behavior.sp, which has not been ported, so it has to keep being defined
by this file; BOMB_HATCH_RANGE_OKAY is read by nobody and stays because the port
is behaviour identical and deleting a constant is a change like any other.
*/
//
//nolint:unused // both are read by SourcePawn the port has not reached
const (
	//sp:name BOMB_HATCH_RANGE_OKAY
	hatchRangeOkay = 5000.0
	//sp:name BOMB_HATCH_RANGE_CRITICAL
	hatchRangeCritical = 1000.0
	//sp:name BOMB_GUARD_RADIUS
	guardRadius = 400.0
)

// maxWatchRadius is how close a friendly sentry has to be for it to be watching
// the bomb already. The shipped code declares it inside IsPossible.
const maxWatchRadius = 1000.0

// OnStart aims the path and tells the team where the bot is holding.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	engine.SpeakConceptIfAllowed(actor, engine.ConceptSentryHere())

	return engine.Continue()
}

// Update holds the ground, closes in when the bot only has a melee or a
// flamethrower, and gives the fight up for a tank or a carrier.
func Update(actor int32) engine.Outcome {
	switch engine.PlayerClass(actor) {
	case engine.ClassSoldier(), engine.ClassPyro(), engine.ClassDemoMan():
		// Tank is more important
		if engine.AttackTankSelectTarget(actor) {
			return engine.ChangeTo(engine.AttackTank(), "Tank inbound")
		}
	}

	flag := engine.BombNearestToHatch()

	if flag == -1 {
		return engine.Done("No bomb")
	}

	if engine.OwnerEntity(flag) != -1 {
		// Someone picked up the bomb!
		return engine.ChangeTo(engine.DefenderAttack(), "Bomb is taken")
	}

	myBot := engine.NextBotOf(actor)
	bombPosition := engine.WorldSpaceCenter(flag)
	myWeapon := engine.ActiveWeapon(actor)

	// Close-range has to get up and personal with them
	if myWeapon != -1 && (engine.WeaponID(myWeapon) == engine.WeaponFlamethrower() || engine.IsMeleeWeapon(myWeapon)) {
		nearest := engine.EnemyPlayerNearestToPosition(actor, bombPosition, guardRadius)

		if nearest != -1 {
			if engine.RepathTime(actor) <= engine.GameTime() {
				engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.5, 1.0))
				engine.RepathToTarget(actor, myBot, nearest)
			}

			engine.PathOf(actor).Update(myBot)

			return engine.Continue()
		}
	}

	/* Guard from the dispenser when there is one on this ground and the bot has a reason to want
	it. Same bomb, same fight, and he heals and reloads without walking away from either */
	var guardPosition [3]float32
	guardPosition = bombPosition

	if engine.Feature(engine.FeatureDispenserGuard()) && engine.WantsDispenser(actor) {
		dispenser := engine.FindFriendlyDispenserNear(actor, bombPosition)

		if dispenser != -1 {
			guardPosition = engine.AbsOriginOf(dispenser)
		}
	}

	// Move towards the ground we are holding if we're too far or can't see the bomb
	if myBot.IsRangeGreaterThanEx(guardPosition, guardRadius) || !engine.IsLineOfFireClearPosition(actor, engine.EyePosition(actor), bombPosition) {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
			engine.RepathToPos(actor, myBot, guardPosition)
		}

		engine.PathOf(actor).Update(myBot)
	}

	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat != 0 {
		engine.EquipBestWeaponForThreat(actor, threat)
	}

	return engine.Continue()
}

// IsPossible says whether this is worth starting: not for a scout or a medic,
// not without a bomb on the floor, not if a sentry already watches it, and not
// if somebody else is already doing it.
//
//sp:name CTFBotCampBomb_IsPossible
func IsPossible(client int32) bool {
	switch engine.PlayerClass(client) {
	case engine.ClassScout(), engine.ClassMedic():
		// We're not very useful for this
		return false
	}

	flag := engine.BombNearestToHatch()

	if flag == -1 {
		return false
	}

	if engine.OwnerEntity(flag) != -1 {
		// No point in camping since DefenderAttack goes for the bomb carrier
		return false
	}

	bombPosition := engine.WorldSpaceCenter(flag)

	iEnt := int32(-1)

	for {
		iEnt = engine.FindEntityByClassname(iEnt, "obj_sentrygun")
		if iEnt == -1 {
			break
		}
		if engine.EntityTeamNumber(iEnt) != engine.GetClientTeam(client) {
			continue
		}

		if engine.VectorDistance(bombPosition, engine.WorldSpaceCenter(iEnt)) <= maxWatchRadius {
			// There;s a sentry watching the bomb
			return false
		}
	}

	// There;s too many of us doing this behavior
	if engine.CountOfBotsWithNamedAction("DefenderCampBomb") > 0 {
		return false
	}

	return true
}
