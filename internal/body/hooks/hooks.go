/*
Package hooks is the game-facing edge of source/redbots3/nextbot_behavior.sp:
the callbacks the actions extension hands to the game's own behaviours.

Four of them exist only to refuse: a defender bot has no business fetching the
flag, idling as a robot engineer, or leaving the spawn room like a robot spy, so
the action ends the moment it starts. Everything else in the file decides; these
five say no and get out of the way.
*/
package hooks

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
MainActionUpdate is the top of every bot's stack, and the only thing the mod
wants from it is the fault injector's hook.

Emptying a stack on purpose is how the idle watchdog gets tested: a bot with no
behaviour is exactly the one nothing else notices.
*/
//
//sp:name CTFBotMainAction_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's, and this reads neither
func MainActionUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if engine.DefenderBotFlag(actor) && engine.ShouldEmptyStack(actor) {
		return action.EndWith("DebugFaults: emptying the stack")
	}

	return engine.PluginContinue()
}

// FetchFlagOnStart refuses: a defender has no bomb to fetch.
//
//sp:name CTFBotFetchFlag_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func FetchFlagOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

// MvMEngineerIdleOnStart refuses: our engineer has his own idle.
//
//sp:name CTFBotMvMEngineerIdle_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func MvMEngineerIdleOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

// SpyLeaveSpawnRoomOnStart refuses: our spy leaves under his own lurk.
//
//sp:name CTFBotSpyLeaveSpawnRoom_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func SpyLeaveSpawnRoomOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

/*
SniperLurkUpdate keeps the game's own lurk unless the rifle is gone.

A sniper holding something that is not a rifle is standing at a perch he cannot
use, which is half of mvm-bj8: he fights like anybody else instead.
*/
//
//sp:name CTFBotSniperLurk_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's
func SniperLurkUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	if !engine.CanUsePrimaryWeapon(actor) {
		// Where did my gun go?
		return engine.SuspendFor(engine.DefenderAttack(), "Lost my rifle")
	}

	return engine.PluginContinue()
}

/*
ScenarioMonitorUpdate is where a defender is handed its next behaviour.

Suspend for the action we desire; once it has ended we come back here and
suspend for another one.
*/
//
//sp:name CTFBotScenarioMonitor_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's
func ScenarioMonitorUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return engine.DesiredBotAction(actor, action)
}

/*
MedicHealUpdatePost runs after the game's own heal think rather than instead of
it, so the medic keeps his walking and his output.

The game's answer to having nobody to heal is to fetch the bomb, and the answer
this mod had for that was to fight whatever is on it, which is the same walk
into the middle of the map by a different name. Everything the team is defending
comes to the hatch eventually.

Only his own shopping comes before healing. This was once the whole break, so a
medic spent the upgrade period walking after whoever he had picked; then it was
the other extreme and he stood at the front with nobody in front of the beam.
Buying his upgrades is the one thing nobody else can do for him, and after that
the man he beams is walking to the front regardless.
*/
//
//sp:name CTFBotMedicHeal_UpdatePost
//sp:public
//
//nolint:revive // unused-parameter: the interval is the game's
func MedicHealUpdatePost(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	if result.ResultType() == engine.ChangeToResult() {
		// In mvm mode, medic bots will go for the flag when there's no
		// patient available. Let's be smarter about it instead.
		resultingAction := result.ResultAction()
		name := resultingAction.ActionName()

		if engine.StrEqual(name, "FetchFlag") {
			return engine.SuspendFor(engine.GuardPoint(), "Nothing to heal, so hold the hatch")
		}
	}

	secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if secondary == -1 {
		return engine.SuspendFor(engine.DefenderAttack(), "No medigun")
	}

	if engine.AttackUberIsPossible(actor, secondary) {
		return engine.SuspendFor(engine.AttackUber(), "Seek uber")
	}

	if engine.MedicReviveIsPossible(actor) {
		return engine.SuspendFor(engine.MedicRevive(), "Revive teammate")
	}

	if engine.RoundState() == engine.RoundStateBetweenRounds() && !engine.ShoppedThisBreak(actor) {
		return engine.PluginContinue()
	}

	if engine.Feature(engine.FeatureMedicPocketsBiggest()) {
		engine.PointMedicAtBiggestBodyNow(action, actor)
	}

	myWeapon := engine.ActiveWeapon(actor)

	if myWeapon != -1 && engine.WeaponID(myWeapon) == engine.WeaponMedigun() {
		engine.MedicUberAndResistNow(actor, myWeapon, action.HandleEntity(engine.ActionHealPatientOffset()))
	}

	return engine.PluginContinue()
}

/*
SniperLurkSelectMoreDangerousThreat is what a lurking sniper shoots first.

Two things outrank everything else and both are about who dies next: an enemy
sniper, who is aiming back down the same sightline, and a medic who is either
healing or charged. Neither is worth picking without line of fire, because a
sniper who has picked a target he cannot see stops looking for one he can.

Nothing else gets an opinion: the answer is null, which sends the choice back to
the normal threat targeting.
*/
//
//sp:name CTFBotSniperLurk_SelectMoreDangerousThreat
//sp:public
//
//nolint:revive // unused-parameter: the action, the bot and the entity are the game's
func SniperLurkSelectMoreDangerousThreat(action engine.Behaviour, nextbot engine.Bot, entity int32, threat1 engine.Known, threat2 engine.Known) (result engine.Outcome, knownEntity engine.Known) {
	me := engine.Actor()

	if !engine.DefenderBotFlag(me) {
		return engine.PluginContinue(), knownEntity
	}

	// Return NULL so the normal threat targeting happens.
	knownEntity = engine.NoKnownEntity()

	iThreat1 := threat1.Entity()

	if engine.IsPlayer(iThreat1) && engine.IsLineOfFireClearEntity(me, engine.EyePosition(me), iThreat1) {
		enemyWeapon := engine.ActiveWeapon(iThreat1)

		if enemyWeapon != -1 {
			enemyWepID := engine.WeaponID(enemyWeapon)

			if engine.WeaponIDIsSniperRifle(enemyWepID) {
				// This sniper ain't gonna snipe me.
				return engine.Changed(), threat1
			} else if enemyWepID == engine.WeaponMedigun() {
				if engine.EntPropEnt(enemyWeapon, engine.PropSend(), "m_hHealingTarget") != -1 || engine.EntPropFloatOf(enemyWeapon, engine.PropSend(), "m_flChargeLevel") >= 1.0 {
					// Healers should die first, ideally before they pop.
					return engine.Changed(), threat1
				}
			}
		}
	}

	iThreat2 := threat2.Entity()

	if engine.IsPlayer(iThreat2) && engine.IsLineOfFireClearEntity(me, engine.EyePosition(me), iThreat2) {
		enemyWeapon := engine.ActiveWeapon(iThreat2)

		if enemyWeapon != -1 {
			enemyWepID := engine.WeaponID(enemyWeapon)

			if engine.WeaponIDIsSniperRifle(enemyWepID) {
				return engine.Changed(), threat2
			} else if enemyWepID == engine.WeaponMedigun() {
				if engine.EntPropEnt(enemyWeapon, engine.PropSend(), "m_hHealingTarget") != -1 || engine.EntPropFloatOf(enemyWeapon, engine.PropSend(), "m_flChargeLevel") >= 1.0 {
					return engine.Changed(), threat2
				}
			}
		}
	}

	return engine.Changed(), knownEntity
}

/*
MainActionShouldAttack lets a defender shoot from anywhere.

The game's rule is written for the invaders, who are not supposed to fire out of
their own spawn. A defender in his spawn shooting into the yard is the whole
point of standing there.
*/
//
//sp:name CTFBotMainAction_ShouldAttack
//
//nolint:revive // unused-parameter: the action, the bot and the threat are the game's
func MainActionShouldAttack(action engine.Behaviour, nextbot engine.Bot, knownEntity engine.Known) (outcome engine.Outcome, result engine.Answer) {
	me := engine.Actor()

	if !engine.DefenderBotFlag(me) {
		return engine.PluginContinue(), result
	}

	// Always attack even in spawn room because we are not the invaders.
	return engine.Changed(), engine.AnswerYes()
}

/*
TacticalMonitorUpdate is the busiest of the game's own behaviours, and the mod
overrides all of it.

Readiness first, before any of the returns below: the upgrade-zone branch used
to return before it ever ran, so a bot shopping at the station had no readiness
of its own and the human mirror never applied. Reported as an engineer beside
spawn who never pressed F4 while a player on RED had; the station is next to
spawn on Decoy, which is where that bot was standing.

Refusing a decision the game is about to make is done by starting its own
timer: FindNearbyTeleporter returns null while its timer runs, and there is no
hook for it.

The order inside the round is deliberate. The buster is above health and ammo
because a bot walking to a health pack through the blast is a bot that arrives
dead. The stickies are pressed here rather than in the attack behaviour because
this holds whatever the bot is doing: a demoman walking to the next fight is
still standing over his last one.
*/
//
//sp:name CTFBotTacticalMonitor_Update
//sp:public
//
//nolint:revive // unused-parameter: the result is the game's
func TacticalMonitorUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	engine.UpdateDefenderReadiness(actor)

	if engine.IsInUpgradeZone(actor) && engine.LookupEntityActionByName(actor, "DefenderUpgrade") != engine.InvalidAction() {
		iClass := engine.PlayerClass(actor)

		if iClass == engine.ClassDemoMan() || iClass == engine.ClassScout() {
			pOpportunisticTimer := engine.CountdownAt(engine.OpportunisticTimer(actor))

			if pOpportunisticTimer.Address() != engine.NoAddress() {
				// We don't do any of these things while upgrading.
				pOpportunisticTimer.Start(interval)
			}
		}

		return engine.PluginContinue()
	}

	if !engine.ShouldUseTeleporterNow(actor) {
		pFindTeleporterTimer := engine.CountdownAt(engine.Address(int32(action) + engine.FindTeleporterOffset()))

		if pFindTeleporterTimer.Address() != engine.NoAddress() {
			// Don't look for any nearby teleporters to use. This forces
			// CTFBotTacticalMonitor::FindNearbyTeleporter to return NULL.
			pFindTeleporterTimer.Start(interval)
		}
	}

	engine.UpdateStuckWatchdog(actor)

	if engine.RoundState() == engine.RoundStateRunning() {
		if engine.EvadeBusterIsPossible(actor) {
			return engine.SuspendFor(engine.EvadeBuster(), "Sentry buster")
		}

		engine.UpdateScoutCombatJump(actor)

		if engine.ShouldDetonateStickies(actor) {
			engine.PressAltFireButton(actor)
		}

		engine.UpdateSpyIntel(actor)

		if engine.SpyCheckIsPossible(actor) {
			return engine.SuspendFor(engine.SpyCheck(), "Spy check")
		}

		lowHealth := false

		healthRatio := engine.HealthRatio(actor)

		if (engine.TimeSinceWeaponFired(actor) > 2.0 || engine.PlayerClass(actor) == engine.ClassSniper()) && healthRatio < engine.HealthCriticalRatio().Float() {
			lowHealth = true
		} else if healthRatio < engine.HealthOkRatio().Float() {
			lowHealth = true
		}

		if lowHealth && engine.ShouldLeaveToBePatchedUp(actor, healthRatio) && engine.GetHealthIsPossible(actor) {
			return engine.SuspendFor(engine.GetHealth(), "Getting health")
		}

		primary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotPrimary())

		if primary != -1 && engine.WeaponID(primary) == engine.WeaponFlamethrower() && (engine.IsCritBoosted(actor) || engine.IsPlayerInCondition(actor, engine.ConditionCritMmmph())) {
			// Don't bother going for ammo while using crits unless our
			// weapon has completely run out.
			if !engine.HasAmmo(primary) && engine.GetAmmoIsPossible(actor) {
				return engine.SuspendFor(engine.GetAmmo(), "Get ammo for crit")
			}
		} else if engine.IsAmmoLowNow(actor) && engine.ShouldLeaveToBePatchedUp(actor, healthRatio) && engine.GetAmmoIsPossible(actor) {
			// Go for ammo when we're low and nearby packs are available.
			return engine.SuspendFor(engine.GetAmmo(), "Getting ammo")
		}
	}

	return engine.PluginContinue()
}

/*
MainActionSelectMoreDangerousThreat is which of two robots the bot fears.

Flamethrowers and melee take the closest, because reach is the whole argument.
One visible threat wins outright. The minigun has two rules of its own: rage
knockback goes on the bomb carrier or a giant, and tanks come last because the
minigun does a quarter damage to them.

Otherwise it is the priority table, and every guide written about this mode says
the same order: the Medic first because a giant being healed cannot be killed at
all, then the Sniper and the Engineer because they are the two the rest of the
team cannot reach, then giants, then whoever is holding the bomb. A robot close
enough to be killing the bot outranks all of it, because a priority target is
worth nothing to a corpse.

Whatever wins, the beam on it wins instead: killing the healer is killing both.
*/
//
//sp:name CTFBotMainAction_SelectMoreDangerousThreat
//sp:public
//
//nolint:revive,gocritic // unused-parameter, ifElseChain: the action and the entity are the game's, and the chain is the shipped shape
func MainActionSelectMoreDangerousThreat(action engine.Behaviour, nextbot engine.Bot, entity int32, threat1 engine.Known, threat2 engine.Known) (result engine.Outcome, knownEntity engine.Known) {
	me := engine.Actor()

	if !engine.DefenderBotFlag(me) {
		return engine.PluginContinue(), knownEntity
	}

	myWeapon := engine.ActiveWeapon(me)

	if myWeapon != -1 && (engine.WeaponID(myWeapon) == engine.WeaponFlamethrower() || engine.IsMeleeWeapon(myWeapon)) {
		// Always target the closest one to us with these weapons.
		return engine.Changed(), engine.HealerOrThreat(nextbot, engine.SelectCloserThreatOf(nextbot, threat1, threat2))
	}

	iThreat1 := threat1.Entity()
	iThreat2 := threat2.Entity()

	// If we can only see one threat, then it's our best target.
	oneVisible := engine.FindOnlyOneVisibleEntity(me, iThreat1, iThreat2)

	if oneVisible == iThreat1 {
		return engine.Changed(), engine.HealerOrThreat(nextbot, threat1)
	}

	if oneVisible == iThreat2 {
		return engine.Changed(), engine.HealerOrThreat(nextbot, threat2)
	}

	if myWeapon != -1 && engine.WeaponID(myWeapon) == engine.WeaponMinigun() {
		if engine.IsRageDraining(me) {
			// When using knockback rage, focus only on particular threats.
			if engine.IsPlayer(iThreat1) && (engine.HasTheFlag(iThreat1) || engine.IsMiniBoss(iThreat1)) {
				return engine.Changed(), threat1
			}

			if engine.IsPlayer(iThreat2) && (engine.HasTheFlag(iThreat2) || engine.IsMiniBoss(iThreat2)) {
				return engine.Changed(), threat2
			}
		}

		// Minigun deals 75% less damage against tanks so prioritize them
		// least.
		if engine.IsBaseBoss(iThreat1) && !engine.IsBaseBoss(iThreat2) {
			return engine.Changed(), threat2
		}

		if !engine.IsBaseBoss(iThreat1) && engine.IsBaseBoss(iThreat2) {
			return engine.Changed(), threat1
		}
	}

	rangeSq1 := nextbot.RangeSquaredTo(iThreat1)
	rangeSq2 := nextbot.RangeSquaredTo(iThreat2)

	generated := engine.Feature(engine.FeatureGeneratedThreatPriority())

	priority1 := engine.ChooseInt(generated, engine.ThreatPriorityGenerated(iThreat1, rangeSq1), engine.ThreatPriority(iThreat1, rangeSq1))
	priority2 := engine.ChooseInt(generated, engine.ThreatPriorityGenerated(iThreat2, rangeSq2), engine.ThreatPriority(iThreat2, rangeSq2))

	if generated {
		engine.ThreatPortAudit(iThreat1, rangeSq1)
		engine.ThreatPortAudit(iThreat2, rangeSq2)
	}

	if engine.Feature(engine.FeatureThreatPriority()) && priority1 != priority2 {
		knownEntity = engine.ChooseThreat(priority1 > priority2, threat1, threat2)
	} else if rangeSq1 < rangeSq2 {
		// Target the closest visible.
		knownEntity = threat1
	} else {
		knownEntity = threat2
	}

	// Target the healer.
	knownEntity = engine.HealerOrThreat(nextbot, knownEntity)

	return engine.Changed(), knownEntity
}

/*
MainActionSelectTargetPoint aims for the game where the game's own answer is
wrong.

Six cases, and each is a different reason. The pipe and sticky launchers get a
full ballistic lead, because a TFBot cannot compensate an arc whose projectile
speed it does not know. The Cow Mangler gets a plain lead, because Valve left
the prediction out of its code entirely, and it falls back to the body when the
led point has no line of fire. The rocket launcher aims at the feet when there
is a crowd standing in the splash: Valve aims at the middle of the robot, which
is right for one robot and wrong for a line of them walking a choke. The sniper
rifles and the Ambassador look up the head bone. The flamethrower aims above a
tank, because flames rise and a Pyro at the treads puts half of every puff into
the ground.
*/
//
//sp:name CTFBotMainAction_SelectTargetPoint
//sp:public
//
//nolint:revive,gocritic // unused-parameter, assignOp: the action and the bot are the game's, and the long form is the shipped one
func MainActionSelectTargetPoint(action engine.Behaviour, nextbot engine.Bot, entity int32) (result engine.Outcome, vec [3]float32) {
	me := engine.Actor()

	if !engine.DefenderBotFlag(me) {
		return engine.PluginContinue(), vec
	}

	myWeapon := engine.ActiveWeapon(me)

	if myWeapon != -1 {
		switch engine.WeaponID(myWeapon) {
		case engine.WeaponGrenadeLauncher(), engine.WeaponPipebombLauncher():
			// TFBots can't compensate their arc if projectile speed differs,
			// so we do our own calculation here.
			var targetPoint [3]float32

			targetPoint = engine.WorldSpaceCenter(entity)
			vecTarget := engine.AbsOriginOf(entity)
			vecActor := engine.Origin(me)

			distance := engine.VectorDistance(vecTarget, vecActor)

			if distance > 150.0 {
				distance = distance / engine.ProjectileSpeed(myWeapon)

				absVelocity := engine.EntityOf(entity).AbsVelocity()

				targetPoint[0] = vecTarget[0] + absVelocity[0]*distance
				targetPoint[1] = vecTarget[1] + absVelocity[1]*distance
				targetPoint[2] = vecTarget[2] + absVelocity[2]*distance
			} else {
				targetPoint = engine.WorldSpaceCenter(entity)
			}

			vecToTarget := engine.SubtractVectors(targetPoint, vecActor)

			a5, unit := engine.NormalizeVector(vecToTarget)
			_ = unit

			ballisticElevation := 0.0125 * a5

			if ballisticElevation > 45.0 {
				ballisticElevation = 45.0
			}

			elevation := ballisticElevation * (engine.Pi() / 180.0)
			sineValue := engine.Sine(elevation)
			cosineValue := engine.Cosine(elevation)

			if cosineValue != 0.0 {
				targetPoint[2] += (sineValue * a5) / cosineValue
			}

			return engine.Changed(), targetPoint
		case engine.WeaponParticleCannon():
			// TFBots won't do projectile prediction with the Cow Mangler 5000
			// since it's left out of the code, so we'll do it ourselves.
			var targetPoint [3]float32

			vecTarget := engine.AbsOriginOf(entity)
			vecActor := engine.AbsOriginOf(me)

			distance := engine.VectorDistance(vecTarget, vecActor)

			if distance > 150.0 {
				distance = distance * 0.00090909092

				absVelocity := engine.EntityOf(entity).AbsVelocity()

				targetPoint[0] = vecTarget[0] + absVelocity[0]*distance
				targetPoint[1] = vecTarget[1] + absVelocity[1]*distance
				targetPoint[2] = vecTarget[2] + absVelocity[2]*distance

				if !engine.IsLineOfFireClearPosition(me, engine.EyePosition(me), targetPoint) {
					vecTarget = engine.WorldSpaceCenter(entity)

					targetPoint[0] = vecTarget[0] + absVelocity[0]*distance
					targetPoint[1] = vecTarget[1] + absVelocity[1]*distance
					targetPoint[2] = vecTarget[2] + absVelocity[2]*distance
				}
			} else {
				targetPoint = engine.WorldSpaceCenter(entity)
			}

			return engine.Changed(), targetPoint
		case engine.WeaponRocketLauncher():
			if engine.IsPlayer(entity) && engine.ShouldAimRocketsAtFeet(me, entity, engine.WeaponRocketLauncher()) {
				return engine.Changed(), engine.AbsOriginOf(entity)
			}
		case engine.WeaponSniperrifle(), engine.WeaponSniperrifleDecap(), engine.WeaponSniperrifleClassic():
			// For sniper rifles, try to look up their head bone to aim at.
			bone := engine.LookupBone(entity, "bip_head")

			if bone != -1 {
				head, vEmpty := engine.BonePosition(entity, bone)
				_ = vEmpty
				head[2] += 3.0

				return engine.Changed(), head
			}

			// Otherwise TFBots aim at the eye position on harder difficulties.
		case engine.WeaponRevolver():
			// Try to aim for the head with the Ambassador.
			if engine.CanRevolverHeadshot(myWeapon) {
				bone := engine.LookupBone(entity, "bip_head")

				if bone != -1 {
					head, vEmpty := engine.BonePosition(entity, bone)
					_ = vEmpty
					head[2] += 3.0

					return engine.Changed(), head
				}

				return engine.Changed(), engine.EyePosition(entity)
			}
		case engine.WeaponFlamethrower():
			if engine.IsBaseBoss(entity) {
				return engine.Changed(), engine.FlameThrowerAimForTank(entity)
			}
		}
	}

	// Let the game do its default aiming.
	return engine.PluginContinue(), vec
}
