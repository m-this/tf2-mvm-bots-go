/*
Package spylurk is source/redbots3/behavior/spylurk.sp.

The spy's default: circle whoever is nearest, get behind him, stab. Tenth
behaviour across, and the one that shows what a behaviour looks like once the
ones it hands off to are already ported: the two sappers are body externs rather
than plugin ones.

//sp:action DefenderSpyLurk CTFBotSpyLurkMvM static
*/
package spylurk

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// behindTolerance is how far behind counts as behind. The shipped code declares
// it where it is used.
const behindTolerance = 0.0

// circleStrafeRange is how close the spy has to be before it starts circling.
const circleStrafeRange = 250.0

// OnStart aims both paths and forgets whoever the last target was.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))
	engine.ChasePathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// Track current target for IsHindrance
	engine.SetAttackTarget(actor, -1)

	return engine.Continue()
}

// Update is the whole behaviour: sap if there is anything to sap, otherwise
// circle and stab, otherwise wander around the bomb.
func Update(actor int32) engine.Outcome {
	if engine.SpySapPlayersSelectTarget(actor) {
		return engine.SuspendFor(engine.SpySapPlayers(), "Sapping player")
	}

	if engine.SpySapSelectTarget(actor) {
		return engine.SuspendFor(engine.SpySap(), "Sapping building")
	}

	myBot := engine.NextBotOf(actor)
	target := engine.BestTargetForSpy(actor, 2000.0)

	if target != -1 {
		if engine.IsStealthed(actor) || engine.IsFeignDeathReady(actor) {
			engine.PressAltFireButton(actor)
		}

		melee := engine.PlayerWeaponSlot(actor, engine.WeaponSlotMelee())

		if melee != -1 {
			engine.SetPlayerActiveWeapon(actor, melee)
		}

		playerThreatForward := engine.EyeVectors(target)
		toPlayerThreat := engine.Origin(target)
		myOrigin := engine.Origin(actor)

		toPlayerThreat = engine.SubtractVectors(toPlayerThreat, myOrigin)

		threatRange, toPlayerThreat := engine.NormalizeVector(toPlayerThreat)
		isBehindVictim := engine.VectorDotProduct(playerThreatForward, toPlayerThreat) > behindTolerance
		isMovingTowardsVictim := true

		if engine.IsLineOfFireClearEntity(actor, engine.EyePosition(actor), target) {
			if threatRange < circleStrafeRange {
				engine.AimHeadTowards(myBot.Body(), engine.WorldSpaceCenter(target), engine.AimMandatory(), 0.1, engine.NoAddress(), "Aim stab")

				if !isBehindVictim {
					// Try to circle around the enemy
					myForward := engine.EyeVectors(actor)
					cross := engine.VectorCrossProduct(playerThreatForward, myForward)

					if cross[2] < 0.0 {
						engine.ExtraButtonsOf(actor).PressButtons(engine.InMoveRight(), 0.1)
					} else {
						engine.ExtraButtonsOf(actor).PressButtons(engine.InMoveLeft(), 0.1)
					}

					// Don't bump into them unless we're going for the stab
					if threatRange < 100.0 && !engine.HasBackstabPotential(target) {
						isMovingTowardsVictim = false
					}
				}
			}

			if threatRange < StabRangeForTarget(target) {
				if engine.IsPlayerInCondition(actor, engine.ConditionDisguised()) {
					if engine.BackstabSkill().Int() == 1 {
						// Attack if we know we can land a backstab
						if engine.EntProp(melee, engine.PropSend(), "m_bReadyToBackstab") != 0 {
							engine.PressFireButton(actor)
						}
					} else {
						// Attack if we think we can land a backstab
						if isBehindVictim || engine.HasBackstabPotential(target) {
							engine.PressFireButton(actor)
						}
					}
				} else {
					// We're exposed anyways, attack!
					engine.PressFireButton(actor)
				}
			}
		}

		if isMovingTowardsVictim {
			engine.ChasePathOf(actor).UpdateChase(myBot, target)
		}
	} else {
		// Can't find anyone near me, just wander around the bomb
		flag := engine.BombNearestToHatch()

		if flag != -1 {
			bombPosition := engine.AbsOriginOf(flag)

			if myBot.IsRangeGreaterThanEx(bombPosition, 200.0) {
				if engine.RepathTime(actor) <= engine.GameTime() {
					engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.9, 1.0))
					engine.RepathToPos(actor, myBot, bombPosition)
				}

				engine.PathOf(actor).Update(myBot)
			}
		}
	}

	engine.SetAttackTarget(actor, target)

	return engine.Continue()
}

// ShouldAttack says no: a spy that opens fire has stopped being a spy.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldAttack(nextbot engine.Bot, knownEntity engine.Known) (changed engine.Outcome, result engine.Answer) {
	// Don't as we will just make ourselves look stupid
	return engine.Changed(), engine.AnswerNo()
}

// IsHindrance stops the spy avoiding people once it is closing on its target.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func IsHindrance(nextbot engine.Bot, entity int32) (changed engine.Outcome, result engine.Answer) {
	me := engine.Actor()

	if engine.AttackTargetOf(me) != -1 && nextbot.IsRangeLessThan(engine.AttackTargetOf(me), 300.0) {
		// Don't avoid anyone as we get closer to our target
		return engine.Changed(), engine.AnswerNo()
	}

	return engine.Changed(), engine.AnswerUndefined()
}

// StabRangeForTarget is longer for a giant, because the model is bigger.
//
//sp:name GetStabRangeForTarget
func StabRangeForTarget(target int32) float32 {
	return 75.0 * engine.ModelScale(target)
}
