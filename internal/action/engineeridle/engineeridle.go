/*
Package engineeridle is source/redbots3/behavior/engineeridle.sp.

What an engineer does when he is not building something: hold the nest, repair
what is on it, carry it away from a buster, move it forward when the front does,
and suspend into the four build behaviours in the order the nest wants them.

Command_DumpNest and DumpBuilding stay hand-written in
source/redbots3/nest_dump.sp: they format into buffers a caller sized and print
them, which is the same gap the teleporter's last result left. mvm-z83 carries
it.

//sp:action DefenderEngineerIdle CTFBotMvMEngineerIdle static
*/
package engineeridle

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// The shipped file declares this and nothing reads it, here or anywhere else.
// Dropping it would be a tidy riding along with a port, so it stays.
//
//sp:name SENTRY_WATCH_BOMB_RANGE
//nolint:unused // the shipped file declares it unused, and the port keeps what ships
const watchBombRange = 400.0

//sp:name m_ctSentrySafe
var sentrySafe [slots.Count]float32

/*
How much closer to the bomb the new ground has to be before the sentry is worth carrying

A sentry in a toolbox shoots nothing, and the walk there and back is most of a
wave's opening. So the candidate has to be meaningfully further forward, not
merely different: without a margin, two areas a few units apart trade places for
ever and the engineer spends the wave carrying.

The cooldown is the second half of the same guard. Having moved, he holds what he
has long enough to have been worth moving.
*/
const (
	//sp:name NEST_ADVANCE_MARGIN
	advanceMargin = 600.0
	//sp:name NEST_ADVANCE_COOLDOWN
	advanceCooldown = 45.0
	//sp:name NEST_ADVANCE_RECHECK
	advanceRecheck = 10.0
)

//sp:name m_ctAdvanceAgain
var advanceAgain [slots.Count]float32

// IsWorthAdvancingTo says the candidate is further along the bomb's route by a
// clear margin, which is what stops him trading two areas for ever.
//
//sp:name IsWorthAdvancingTo
func IsWorthAdvancingTo(held engine.Area, candidate engine.Area) bool {
	if candidate == engine.NullArea() || candidate == held {
		return false
	}

	if held == engine.NullArea() {
		return true
	}

	// Travel distance to where the bomb is going, so "forward" means along the route and not through a wall
	return engine.TravelDistanceToBombTarget(engine.NavArea(candidate)) <
		engine.TravelDistanceToBombTarget(engine.NavArea(held))-advanceMargin
}

var (
	//sp:name m_ctSentryCooldown
	sentryCooldown [slots.Count]float32
	//sp:name m_ctDispenserSafe
	dispenserSafe [slots.Count]float32
	//sp:name m_ctDispenserCooldown
	dispenserCooldown [slots.Count]float32
	//sp:name m_ctFindNestHint
	findNestHint [slots.Count]float32
	//sp:name m_ctAdvanceNestSpot
	advanceNestSpot [slots.Count]float32
	//sp:name m_ctRecomputePathMvMEngiIdle
	recomputePath [slots.Count]float32
	//sp:name g_bGoingToGrabBuilding
	goingToGrab [slots.Count]bool
	//sp:name m_hBuildingToGrab
	buildingToGrab [slots.Count]int32
	/* The nest an engineer was holding before a buster moved him off it, NULL_AREA when he is on it

	The shipped declaration fills this with NULL_AREA. NULL_AREA is zero and a SourcePawn global
	starts at zero, so the fill says nothing the declaration does not. */
	//
	//sp:name m_aNestAreaBeforeHaul
	nestBeforeHaul [slots.Count]engine.Area
	// The nest an engineer is leaving for better ground, NULL_AREA when no relocation haul is running
	//
	//sp:name m_aNestAreaBeforeRelocate
	nestBeforeRelocate [slots.Count]engine.Area
	// When a relocation haul stops being worth finishing
	//
	//sp:name m_ctNestRelocateDeadline
	relocateDeadline [slots.Count]float32
)

/*
How long an engineer gets to move a sentry to better ground before he puts it down where he is

The move is decided between waves and the wave can start while he is still
walking. A sentry in a toolbox when the robots arrive is worse than a badly placed
level three, so the haul runs on a clock rather than on a promise that it finishes
in time.
*/
//
//sp:name NEST_RELOCATE_HAUL_TIME
const relocateHaulTime = 20.0

/*
How long an engineer may walk around holding a building before he puts it down

The carry had no clock at all. He only tries to place while he is within seventy
units of the nest centre, so a centre he cannot reach, or a spot that keeps
refusing the placement, leaves him holding it for the rest of the mission.
Reported from play on Coal Town, mid wave and between waves both, and the mid wave
one costs the team its sentry for the whole wave.

Twenty five seconds is longer than any haul this mod starts on purpose and shorter
than a wave. Down where he stands beats carried, which is the same answer the
relocation timeout already gives: the ground under his feet is at worst ground he
was already crossing.
*/
//
//sp:name CARRY_GIVE_UP_TIME
const carryGiveUpTime = 25.0

var (
	//sp:name m_ctCarryDeadline
	carryDeadline [slots.Count]float32
	//sp:name m_ctSentryUnderFire
	sentryUnderFire [slots.Count]float32
	//sp:name m_iSentryHealthLast
	sentryHealthLast [slots.Count]int32
)

// How long a sentry counts as under fire after the last health it lost
//
//sp:name SENTRY_UNDER_FIRE_TIME
const underFireTime = 3.0

// OnStart forgets everything the last life left behind.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// A fresh engineer holds no ground yet, so there is nowhere for a buster to have moved him off
	nestBeforeHaul[actor] = engine.NullArea()
	nestBeforeRelocate[actor] = engine.NullArea()
	relocateDeadline[actor] = -1.0
	sentryHealthLast[actor] = 0

	ResetProperties(actor)

	return engine.Continue()
}

/*
An engineer with no sentry who is not building one, said out loud every few seconds

Measured on Coaltown: wave four ran for eight minutes and the engineer had a sentry
standing for fifteen percent of it, with one gap of three hundred and eighty five
seconds. No build action failed in that time, because no build action ran. He was
inside this update the whole while, so one of its early returns owns the wave and
there is no way to tell which from outside.

Throttled per engineer, and only while he has nothing standing, so a working nest
is silent.
*/
//
//sp:name ENGINEER_STALL_REPORT
const stallReport = 10.0

//sp:name m_ctEngineerStallReport
var stallReportAt [slots.Count]float32

// ReportEngineerStall says which early return owns the wave.
//
//sp:name ReportEngineerStall
func ReportEngineerStall(actor int32, where string) {
	if stallReportAt[actor] > engine.GameTime() {
		return
	}

	stallReportAt[actor] = engine.GameTime() + stallReport

	engine.PrintToServer("[defenderbots] engineer %N has no sentry at %.1f: %s (nest %s, carrying %s, grabbing %s)",
		actor, engine.GameTime(), where,
		engine.Choose(nestAreaOf(actor) == engine.NullArea(), "none", "held"),
		engine.Choose(engine.IsCarryingObject(actor), "yes", "no"),
		engine.Choose(goingToGrab[actor], "yes", "no"))
}

// nestAreaOf is the slot the rest of the mod keeps this engineer's ground in.
func nestAreaOf(actor int32) engine.Area {
	return engine.NestAreaOf(actor)
}

// Update is the whole of what an engineer does between builds.
func Update(actor int32) engine.Outcome {
	/* Counting what is in his hands, for both

	A carried building is not a reason to build a second one, and answering that question wrong is
	what a play-test filmed: the engineer stood holding a sentry, scrolling through his weapons and
	opening and cancelling the build menu, over and over. Two paths were undoing each other every
	frame. The carry logic equips the wrench and presses fire to put it down; the gate below saw no
	sentry, suspended for the build action, and that equips the toolbox and reopens the menu. */
	sentry := engine.HasObjectOfType(actor, engine.ObjectSentry(), engine.ModeNone())
	dispenser := engine.HasObjectOfType(actor, engine.ObjectDispenser(), engine.ModeNone())

	// Standing, for the question of whether the nest is up: one in his hands is not defending anything
	sentryStanding := engine.ObjectOfType(actor, engine.ObjectSentry())

	stalled := sentryStanding == engine.InvalidEntReference() && engine.RoundState() == engine.RoundStateRunning()

	bShouldAdvance := ShouldAdvanceNestSpot(actor)

	/* A buster is walking at the nest, so the nest stops being where the engineer wants to be
	Handled before the advance below and before anything else this action does, because the whole
	value of it is spending the walk the buster still has left. It reuses the carry machinery that
	advancing already uses: the goal area is somewhere out of the blast rather than a better nest,
	and the engineer picks the sentry up and walks it there the same way */
	buster := int32(-1)

	if !goingToGrab[actor] {
		haul, found := ShouldHaulFromSentryBuster(actor, sentry)

		buster = found

		if haul {
			retreat := engine.PickBusterRetreatArea(sentry, buster)

			if retreat != engine.NullArea() {
				if engine.DebugActions().Bool() {
					engine.PrintToServer("CTFBotMvMEngineerIdle_Update: HAUL FROM BUSTER")
				}

				engine.SpeakConceptIfAllowed(actor, engine.ConceptIncoming())

				/* The nest this engineer is leaving, so that it goes back to it
				Without this the spot the sentry was carried to becomes the nest for the rest of the
				wave, and a spot chosen for being far from one robot is not a spot to hold ground from */
				nestBeforeHaul[actor] = engine.NestAreaOf(actor)

				ResetProperties(actor)

				engine.SetNestArea(actor, retreat)

				goingToGrab[actor] = true
				buildingToGrab[actor] = engine.EntIndexToEntRef(sentry)

				engine.PluginBotOf(actor).SetPathGoalEntity(sentry)

				return engine.Continue()
			}
		}
	}

	/* The buster is gone and the sentry is standing somewhere it was carried to, not somewhere
	chosen to shoot from. Going back is the same walk in the other direction */
	if nestBeforeHaul[actor] != engine.NullArea() && buster == -1 && !goingToGrab[actor] && sentry != engine.InvalidEntReference() {
		home := nestBeforeHaul[actor]

		nestBeforeHaul[actor] = engine.NullArea()

		ResetProperties(actor)

		engine.SetNestArea(actor, home)

		goingToGrab[actor] = true
		buildingToGrab[actor] = engine.EntIndexToEntRef(sentry)

		engine.PluginBotOf(actor).SetPathGoalEntity(sentry)

		return engine.Continue()
	}

	/* The between-waves answer says this nest moved and the buildings are still standing on the old
	ground, which is what happens when nothing tore them down at the upgrade station. Same carry as
	the buster haul above, with better ground as the goal rather than open ground */
	if engine.NestRelocateOf(actor) != engine.NullArea() &&
		nestBeforeHaul[actor] == engine.NullArea() &&
		!goingToGrab[actor] &&
		!engine.IsCarryingObject(actor) &&
		(sentry == engine.InvalidEntReference() || !engine.IsBuildingUp(sentry)) {
		destination := engine.NestRelocateOf(actor)

		// One move per between-waves period, whether or not there is anything to carry to it
		engine.SetNestRelocate(actor, engine.NullArea())

		if sentry == engine.InvalidEntReference() {
			// Nothing to carry, so the new ground is simply where the next sentry goes up
			engine.SetNestArea(actor, destination)
		} else { //nolint:gocritic // elseif: the shipped file nests these, and the port keeps its shape
			if engine.DebugActions().Bool() {
				engine.PrintToServer("CTFBotMvMEngineerIdle_Update: RELOCATE NEST")
			}

			nestBeforeRelocate[actor] = engine.NestAreaOf(actor)

			ResetProperties(actor)

			relocateDeadline[actor] = engine.GameTime() + relocateHaulTime
			engine.SetNestArea(actor, destination)

			goingToGrab[actor] = true
			buildingToGrab[actor] = engine.EntIndexToEntRef(sentry)

			engine.PluginBotOf(actor).SetPathGoalEntity(sentry)

			/* The dispenser stays behind on ground nobody holds any more
			Only one building can be carried at a time and the sentry is the one worth the walk, so
			the dispenser is spent rather than dragged: the idle loop below builds another one at the
			new nest for 100 metal as soon as the sentry is safe */
			engine.DetonateObjectOfType(actor, engine.ObjectDispenser())

			return engine.Continue()
		}
	}

	if bShouldAdvance && !goingToGrab[actor] {
		/* Move, but only to ground that is actually further forward, and only once in a while

		Reported from play on Coaltown: the engineer on the building at the right picks the sentry
		up the moment the wave starts and keeps picking it up, and puts it down where it can see
		nothing.

		Both halves of that are this block. It re-scored the nest on every frame the advance
		condition held and moved to whatever came back, and what came back is not required to be
		any further forward than where he already was: PickBuildArea answers with the best area it
		can find, and if that is the one he is standing on, or another one equally far behind the
		front, the condition is still true next frame and he picks the sentry up again.

		So the candidate has to beat the ground he holds by a clear margin before he commits to
		carrying anything, and having moved, he leaves it alone for a while. A sentry in a toolbox
		shoots nothing, so a move has to buy more than it costs. */
		candidate := engine.PickBuildArea(actor)

		if advanceAgain[actor] > engine.GameTime() || !IsWorthAdvancingTo(engine.NestAreaOf(actor), candidate) {
			// Nothing better within reach, so stop asking for a bit and hold what he has
			advanceNestSpot[actor] = engine.GameTime() + advanceRecheck
		} else {
			if engine.DebugActions().Bool() {
				engine.PrintToServer("CTFBotMvMEngineerIdle_Update: ADVANCE")
			}

			// RIGHT NOW
			ResetProperties(actor)

			engine.SetNestArea(actor, candidate)
			advanceAgain[actor] = engine.GameTime() + advanceCooldown

			if sentry != engine.InvalidEntReference() && engine.NestAreaOf(actor) != engine.NullArea() {
				goingToGrab[actor] = true

				buildingToGrab[actor] = engine.EntIndexToEntRef(sentry)

				engine.PluginBotOf(actor).SetPathGoalEntity(sentry)
			}
		}
	}

	myNextbot := engine.NextBotOf(actor)
	myBody := myNextbot.Body()
	myLoco := myNextbot.Locomotion()

	/* The clock ran out on a relocation, which means the wave started while he was still walking

	Down where he stands beats carried into the middle of a wave: the ground under his feet is at
	worst ground he was already crossing, and the alternative is a nest that exists in a toolbox.
	Before he picks the sentry up there is nothing to put down and the old nest is still a nest */
	if nestBeforeRelocate[actor] != engine.NullArea() && relocateDeadline[actor] > 0.0 && engine.GameTime() > relocateDeadline[actor] {
		relocateDeadline[actor] = -1.0

		if engine.IsCarryingObject(actor) {
			here := engine.NearestNavArea(engine.AbsOriginOf(actor), false, 500.0, false, true, engine.TeamAny())

			if here != engine.NullArea() {
				engine.SetNestArea(actor, here)
			}
		} else {
			engine.SetNestArea(actor, nestBeforeRelocate[actor])

			goingToGrab[actor] = false
			buildingToGrab[actor] = engine.InvalidEntReference()
		}

		nestBeforeRelocate[actor] = engine.NullArea()
	}

	if goingToGrab[actor] {
		building := engine.EntRefToEntIndex(buildingToGrab[actor])

		/* The clock on the carry, started when he first has the thing in his hands

		Placing it needs him within seventy units of the nest centre, and nothing here says what to
		do when he never gets there. */
		//nolint:gocritic // ifElseChain: the shipped file is this chain, and the port keeps its shape
		if !engine.IsCarryingObject(actor) {
			carryDeadline[actor] = 0.0
		} else if carryDeadline[actor] <= 0.0 {
			carryDeadline[actor] = engine.GameTime() + carryGiveUpTime
		} else if engine.GameTime() > carryDeadline[actor] {
			here := engine.NearestNavArea(engine.AbsOriginOf(actor), false, 500.0, false, true, engine.TeamAny())

			carryDeadline[actor] = 0.0

			engine.LogBuildFailure(actor, "carry", "held it too long, putting it down here")

			// The nest is where he is now, so the placing branch below takes it the moment it runs
			if here != engine.NullArea() {
				engine.SetNestArea(actor, here)
			}
		}

		if building == engine.InvalidEntReference() {
			goingToGrab[actor] = false
			buildingToGrab[actor] = engine.InvalidEntReference()
			nestBeforeRelocate[actor] = engine.NullArea()
			relocateDeadline[actor] = -1.0

			if engine.DebugActions().Bool() {
				engine.PrintToServer("CTFBotMvMEngineerIdle_Update: g_bGoingToGrabBuilding : building %i | m_aNestArea %x", building, engine.NestAreaOf(actor))
			}

			engine.DetonateObjectOfType(actor, engine.ObjectSentry())
			engine.DetonateObjectOfType(actor, engine.ObjectDispenser())

			engine.PluginBotOf(actor).SetPathing(false)

			if stalled {
				ReportEngineerStall(actor, "the building he was fetching is gone")
			}

			return engine.Continue()
		}

		engine.UpdateLookAroundForEnemies(actor, false)

		if !engine.IsCarryingObject(actor) {
			flDistanceToBuilding := engine.VectorDistance(engine.AbsOriginOf(actor), engine.AbsOriginOf(building))

			if flDistanceToBuilding < 90.0 {
				engine.EquipWeaponSlot(actor, engine.WeaponSlotMelee())

				engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(building), engine.AimCritical(), 1.0, engine.NoAddress(), "Grab building")
				engine.PressAltFireButton(actor)
			}
		} else { //nolint:gocritic // elseif: the shipped file nests these, and the port keeps its shape
			if engine.NestAreaOf(actor) != engine.NullArea() {
				center := engine.NestAreaOf(actor).Center()
				engine.PluginBotOf(actor).SetPathGoalVector(center)

				flDistanceToGoal := engine.VectorDistance(engine.AbsOriginOf(actor), center)

				if flDistanceToGoal < 200.0 {
					// Crouch when closer than 200 hu
					if !myLoco.IsStuck() {
						engine.ExtraButtonsOf(actor).PressButtons(engine.InDuck(), 0.1)
					}

					if flDistanceToGoal < 70.0 {
						// Try placing building when closer than 70 hu
						objBeingBuilt := engine.CarriedObject(actor)

						if objBeingBuilt == -1 {
							return engine.Continue()
						}

						bPlacementOK := engine.IsPlacementOK(objBeingBuilt)

						engine.PressFireButton(actor)

						if !bPlacementOK && myBody.IsHeadAimingOnTarget() && myBody.HeadSteadyDuration() > 0.6 {
							engine.SetNestArea(actor, engine.PickBuildArea(actor))
						} else {
							goingToGrab[actor] = false
							buildingToGrab[actor] = engine.InvalidEntReference()
							nestBeforeRelocate[actor] = engine.NullArea()
							relocateDeadline[actor] = -1.0

							engine.PluginBotOf(actor).SetPathing(false)
						}
					}
				}
			}
		}

		engine.PluginBotOf(actor).SetPathing(true)

		return engine.Continue()
	}

	if (engine.NestAreaOf(actor) == engine.NullArea() || bShouldAdvance) || sentry == engine.InvalidEntReference() {
		// HasStarted && !IsElapsed
		if findNestHint[actor] > 0.0 && findNestHint[actor] > engine.GameTime() {
			return engine.Continue()
		}

		// Start
		findNestHint[actor] = engine.GameTime() + engine.RandomFloat(1.0, 2.0)

		engine.SetNestArea(actor, engine.PickBuildArea(actor))
	}

	if bShouldAdvance {
		if stalled {
			ReportEngineerStall(actor, "advancing the nest")
		}

		return engine.Continue()
	}

	UpdateSentryUnderFire(actor, sentry)

	if sentry != -1 {
		if (sentrySafe[actor] > engine.GameTime() || sentryUnderFire[actor] > engine.GameTime()) && !goingToGrab[actor] {
			mySecondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

			if mySecondary != -1 && engine.WeaponID(mySecondary) == engine.WeaponLaserPointer() && myNextbot.IsRangeLessThan(sentry, 180.0) {
				threat := myNextbot.Vision().PrimaryKnownThreat(false)

				if threat != 0 {
					iThreat := threat.Entity()

					/* Two reasons to hold the wrangler, and the shield is the one that was missing

					Out of range, the wrangler is the only way the sentry reaches the threat at all,
					which is what this did and all it did. Under fire, it is worth holding for the
					shield alone: two thirds of the damage aimed at the sentry stops there, and a
					sentry that survives the giant is worth more than the seconds of aim it costs.

					Both want the same thing of the bot, so both run the same code: point it at the
					threat and hold the buttons */
					defending := sentryUnderFire[actor] > engine.GameTime()

					if (defending || engine.VectorDistance(engine.AbsOriginOf(sentry), engine.AbsOriginOf(iThreat)) > engine.SentryMaxRange()) && engine.IsLineOfFireClearEntity(actor, engine.EyePosition(actor), iThreat) {
						engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(iThreat), engine.AimMandatory(), 0.1, engine.NoAddress(), "Aiming!")
						engine.SetPlayerActiveWeapon(actor, mySecondary)

						if myBody.IsHeadAimingOnTarget() && engine.EntProp(sentry, engine.PropSend(), "m_bPlayerControlled") != 0 {
							engine.RunScriptCode(actor, engine.Default(), engine.Default(), "self.PressFireButton(0.1);self.PressAltFireButton(0.1)")
						}

						engine.PluginBotOf(actor).SetPathing(false)

						return engine.Continue()
					}
				}
			}
		}
	}

	if engine.NestAreaOf(actor) == engine.NullArea() && stalled {
		ReportEngineerStall(actor, "no nest area to build on")
	}

	if engine.NestAreaOf(actor) != engine.NullArea() {
		if sentry != engine.InvalidEntReference() {
			if IsSentrySafe(sentry) {
				sentrySafe[actor] = engine.GameTime() + 3.0
			}

			sentryCooldown[actor] = engine.GameTime() + 3.0
		} else {
			/* do not have a sentry; retreat for a few seconds if we had a
			 * sentry before this; then build a new sentry */
			if sentryCooldown[actor] >= engine.GameTime() {
				ReportEngineerStall(actor, "waiting out the rebuild cooldown")
			}

			if sentryCooldown[actor] < engine.GameTime() {
				sentryCooldown[actor] = engine.GameTime() + 3.0

				return engine.SuspendFor(engine.BuildSentrygun(), "No sentry - building a new one")
			}
		}

		// Don't build a dispenser if we don't have a sentry...
		if sentry != engine.InvalidEntReference() {
			if dispenser != engine.InvalidEntReference() {
				// sentry is not safe.
				if sentrySafe[actor] < engine.GameTime() {
					dispenserCooldown[actor] = engine.GameTime() + 3.0
				}
			} else {
				/* do not have a dispenser; retreat for a few seconds if we had a
				 * dispenser before this; then build a new dispenser */
				if dispenserCooldown[actor] < engine.GameTime() && sentrySafe[actor] > engine.GameTime() {
					dispenserCooldown[actor] = engine.GameTime() + 3.0

					return engine.SuspendFor(engine.BuildDispenser(), "Sentry safe, No dispenser - building one")
				}
			}
		}
	}

	/* The nest is standing and the wave has not started, so there is time for a teleporter
	Nothing below this point is skipped by it: the action gives up the moment the wave starts or
	the sentry stops standing, and the engineer goes straight back to the nest */
	if sentrySafe[actor] > engine.GameTime() && !goingToGrab[actor] && engine.ShouldBuildTeleporter(actor) {
		return engine.SuspendFor(engine.BuildTeleporter(), "Nest is up, building a teleporter")
	}

	// A second gun beside the first, put there on purpose rather than dropped wherever he was facing
	if sentrySafe[actor] > engine.GameTime() && !goingToGrab[actor] && engine.ShouldBuildDisposable(actor) {
		return engine.SuspendFor(engine.BuildDisposable(), "Nest is up, standing a mini beside it")
	}

	/* The dispenser is only a job once the sentry is not one

	This branch comes first and returns, so an engineer whose dispenser was a level two stood at it
	swinging a wrench while the sentry twenty feet away was being shot to pieces. The sentry is the
	nest; the dispenser is what feeds it. Reported as the engineer not repairing his buildings,
	which he was doing, just never the one being destroyed. */
	sentryWantsMetal := sentry != engine.InvalidEntReference() && SentryNeedsMetal(sentry)

	if dispenser != engine.InvalidEntReference() && !sentryWantsMetal && sentrySafe[actor] > engine.GameTime() {
		if engine.UpgradeLevel(dispenser) < 3 || engine.EntityHealth(dispenser) < engine.EntityMaxHealth(dispenser) {
			dist := engine.VectorDistance(engine.AbsOriginOf(actor), engine.AbsOriginOf(dispenser))

			if recomputePath[actor] < engine.GameTime() {
				recomputePath[actor] = engine.GameTime() + engine.RandomFloat(1.0, 2.0)

				dir := engine.SubtractVectors(engine.AbsAnglesOf(dispenser), engine.AbsOriginOf(actor))
				_, dir = engine.NormalizeVector(dir)

				goal := engine.AbsOriginOf(dispenser)
				goal[0] -= 50.0 * dir[0]
				goal[1] -= 50.0 * dir[1]
				goal[2] -= 50.0 * dir[2]

				if engine.IsPathToVectorPossible(actor, goal) {
					engine.PluginBotOf(actor).SetPathGoalVector(goal)
				} else {
					engine.PluginBotOf(actor).SetPathGoalEntity(sentry)
				}

				engine.PluginBotOf(actor).SetPathing(true)
			}

			if dist < 90.0 {
				if !myLoco.IsStuck() {
					engine.ExtraButtonsOf(actor).PressButtons(engine.InDuck(), 0.1)
				}

				engine.EquipWeaponSlot(actor, engine.WeaponSlotMelee())

				engine.UpdateLookAroundForEnemies(actor, false)

				engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(dispenser), engine.AimCritical(), 1.0, engine.NoAddress(), "Work on my Dispenser")
				engine.PressFireButton(actor)
			}

			return engine.Continue()
		}
	}

	if sentry != engine.InvalidEntReference() {
		dist := engine.VectorDistance(engine.AbsOriginOf(actor), engine.AbsOriginOf(sentry))

		/* A finished sentry is not a job
		The wrench does nothing to a level three at full health and full shells, and the engineer
		swinging it is an engineer not shooting at the robots walking into his nest */
		if !SentryNeedsMetal(sentry) {
			engine.EquipWeaponSlot(actor, engine.WeaponSlotPrimary())

			engine.UpdateLookAroundForEnemies(actor, true)
		} else if CanRepairFromRange(actor, sentry, dist) {
			// The Rescue Ranger repairs from behind cover, which is the whole reason to carry it
			engine.EquipWeaponSlot(actor, engine.WeaponSlotPrimary())

			engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(sentry), engine.AimCritical(), 1.0, engine.NoAddress(), "Repair my Sentry from here")

			if myBody.IsHeadAimingOnTarget() {
				engine.PressFireButton(actor)
			}

			engine.PluginBotOf(actor).SetPathing(false)

			return engine.Continue()
		}

		if recomputePath[actor] < engine.GameTime() {
			recomputePath[actor] = engine.GameTime() + engine.RandomFloat(1.0, 2.0)

			vTurretAngles := engine.TurretAngles(sentry)
			dir := engine.AngleForward(vTurretAngles)

			goal := engine.AbsOriginOf(sentry)
			goal[0] -= 50.0 * dir[0]
			goal[1] -= 50.0 * dir[1]
			goal[2] -= 50.0 * dir[2]

			if engine.IsPathToVectorPossible(actor, goal) {
				engine.PluginBotOf(actor).SetPathGoalVector(goal)
			} else {
				engine.PluginBotOf(actor).SetPathGoalEntity(sentry)
			}

			engine.PluginBotOf(actor).SetPathing(true)
		}

		if dist < 90.0 && SentryNeedsMetal(sentry) {
			if !myLoco.IsStuck() {
				engine.ExtraButtonsOf(actor).PressButtons(engine.InDuck(), 0.1)
			}

			engine.EquipWeaponSlot(actor, engine.WeaponSlotMelee())

			engine.UpdateLookAroundForEnemies(actor, false)

			engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(sentry), engine.AimCritical(), 1.0, engine.NoAddress(), "Work on my Sentry")
			engine.PressFireButton(actor)
		}
	}

	return engine.Continue()
}

/*
UpdateSentryUnderFire is whether the sentry is being shot at, from the only thing
that says so without a damage hook.

Health that went down since the last frame. A repair puts it back up and does not
clear the timer, which is the point: an engineer wrenching a sentry that keeps
losing health is an engineer who should be holding the wrangler instead.
*/
//
//sp:name UpdateSentryUnderFire
func UpdateSentryUnderFire(actor int32, sentry int32) {
	if sentry == engine.InvalidEntReference() {
		sentryHealthLast[actor] = 0
		return
	}

	health := engine.EntityHealth(sentry)

	if sentryHealthLast[actor] > 0 && health < sentryHealthLast[actor] {
		sentryUnderFire[actor] = engine.GameTime() + underFireTime
	}

	sentryHealthLast[actor] = health
}

/*
ShouldHaulFromSentryBuster is whether to pick the sentry up and walk it away from
a buster, and which buster.

Only worth doing while the buster still has ground to cover. Inside flee range the
engineer is better off running than carrying, which is what CTFBotEvadeBuster does
with him, and a buster already detonating cannot be walked away from at all.

A mini sentry is not carried. It costs 100 metal and two seconds, so a Gunslinger
engineer lets the buster have it and builds another one behind the blast.
*/
//
//sp:name ShouldHaulFromSentryBuster
func ShouldHaulFromSentryBuster(actor int32, sentry int32) (haul bool, buster int32) {
	buster = -1

	if sentry == engine.InvalidEntReference() {
		return false, buster
	}

	if engine.RoundState() != engine.RoundStateRunning() {
		return false, buster
	}

	if engine.IsCarryingObject(actor) || engine.IsBuildingUp(sentry) {
		return false, buster
	}

	buster = engine.FindSentryBusterNear(engine.AbsOriginOf(sentry), engine.PlayerEnemyTeam(actor), engine.BusterHaulRange())

	if buster == -1 {
		return false, buster
	}

	if engine.IsMiniBuilding(sentry) {
		return false, buster
	}

	// Close enough that carrying is slower than the buster is
	if engine.VectorDistance(engine.AbsOriginOf(sentry), engine.WorldSpaceCenter(buster)) < engine.BusterFleeRange() {
		return false, buster
	}

	return true, buster
}

/*
SentryNeedsMetal is whether there is anything the wrench can still do for this
sentry.

A mini sentry cannot be upgraded and is rebuilt rather than nursed, so for a
Gunslinger this is only ever about damage taken. Shells are counted because a
sentry out of ammo is a sentry that does nothing while it looks perfectly healthy.
*/
//
//sp:name SentryNeedsMetal
func SentryNeedsMetal(sentry int32) bool {
	if engine.IsBuildingUp(sentry) {
		return true
	}

	if engine.EntityHealth(sentry) < engine.EntityMaxHealth(sentry) {
		return true
	}

	if !engine.IsMiniBuilding(sentry) && engine.UpgradeLevel(sentry) < 3 {
		return true
	}

	return engine.EntProp(sentry, engine.PropSend(), "m_iAmmoShells") < 100
}

/*
How often a bolt was fired at a sentry that then failed to gain any health

A player reported an engineer stood firing Rescue Ranger bolts into a wall with the
sentry behind it. Reproducing that here found nothing, and the reason is worth
keeping: across four waves the state this can happen in, a damaged sentry with its
engineer standing two hundred units or more away, occurred in zero samples out of a
hundred and thirty seven. The sentry is at full health or it is dead, and five
second sampling cannot see a rare state at all.

So this counts the event rather than sampling the state. A counter accumulates, and
sampling a counter loses nothing however rare the thing is: one stall in an hour
still shows up as a one.

It only counts. Deciding what the engineer should do about it needs to know how
often it happens and that is what this is for.
*/
//
//sp:name RANGE_REPAIR_PATIENCE
const rangeRepairPatience = 3.0

var (
	//sp:name m_iRangeRepairHealth
	rangeRepairHealth [slots.Count]int32
	//sp:name m_ctRangeRepairSince
	rangeRepairSince [slots.Count]float32
	//sp:name m_iRangeRepairStalls
	rangeRepairStalls [slots.Count]int32
)

// RangeRepairStallsOf is how many times this engineer's bolts bought nothing.
//
//sp:name RangeRepairStallsOf
func RangeRepairStallsOf(client int32) int32 {
	return rangeRepairStalls[client]
}

// NoteRangeRepair counts a stall once per stall rather than once per frame of one.
//
//sp:name NoteRangeRepair
func NoteRangeRepair(actor int32, sentry int32) {
	health := engine.EntityHealth(sentry)
	now := engine.GameTime()

	// Health went up, or this is the first bolt: either way the clock starts here
	if health > rangeRepairHealth[actor] || rangeRepairSince[actor] <= 0.0 {
		rangeRepairHealth[actor] = health
		rangeRepairSince[actor] = now

		return
	}

	rangeRepairHealth[actor] = health

	if now-rangeRepairSince[actor] < rangeRepairPatience {
		return
	}

	// Counted once per stall rather than once per frame of one
	rangeRepairSince[actor] = now
	rangeRepairStalls[actor]++

	engine.LogBuildFailure(actor, "repairing at range", "three seconds of bolts and the sentry gained nothing")
}

// ForgetRangeRepair starts the count again.
//
//sp:name ForgetRangeRepair
func ForgetRangeRepair(actor int32) {
	rangeRepairHealth[actor] = 0
	rangeRepairSince[actor] = 0.0
}

// CanRepairFromRange says a Rescue Ranger bolt repairs at range, so its engineer
// does not walk into the open to hold a nest.
//
//sp:name CanRepairFromRange
func CanRepairFromRange(actor int32, sentry int32, dist float32) bool {
	if !engine.IsRescueRangerEquipped(actor) {
		return false
	}

	/* A bolt repairs a building and does not reload one

	Only a hit with the wrench puts shells back in a sentry. A sentry at full health with an empty
	magazine still answers yes to SentryNeedsMetal, so the engineer stood at range firing bolts into
	something that was already whole and never gained a shell. Reported from play as Rescue Ranger
	engineers refusing to reload, and measured on Coal Town before the fix: 21 of 126 samples of a
	full health sentry had it under fifty shells, and some at none.

	So range repair is for damage and nothing else. Ammo is a walk. */
	if engine.EntityHealth(sentry) >= engine.EntityMaxHealth(sentry) {
		return false
	}

	// Close enough to swing at, and the wrench repairs faster
	if dist < 200.0 {
		return false
	}

	if dist > engine.SentryMaxRange() {
		return false
	}

	if engine.AmmoOf(actor, engine.PropData(), "m_iAmmo", 3) < 30 {
		return false
	}

	if !engine.IsLineOfFireClearEntity(actor, engine.EyePosition(actor), sentry) {
		ForgetRangeRepair(actor)

		return false
	}

	// Watched, not acted on. What to do about a stall is a separate argument and needs this first
	NoteRangeRepair(actor, sentry)

	return true
}

// OnEnd stops the walking. An engineer should only truly leave this behaviour
// when he dies.
func OnEnd(actor int32) {
	// NOTE: engineer should only truly leave this behavior when he dies, it should otherwise be impossible
	engine.PluginBotOf(actor).SetPathing(false)
}

// OnMoveToSuccess clears the stuck status: because of our constant pathing, we
// are not stuck once we arrive at our desired position.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func OnMoveToSuccess(actor int32, path int32) engine.Outcome {
	// Because of our constant pathing, we are not stuck once we arrive to our desired position
	engine.NextBotOf(actor).Locomotion().ClearStuckStatus("Arrived at goal")

	return engine.TryContinue()
}

// ResetProperties forgets what this engineer was in the middle of.
//
//sp:name CTFBotMvMEngineerIdle_ResetProperties
func ResetProperties(actor int32) {
	ForgetRangeRepair(actor)

	buildingToGrab[actor] = engine.InvalidEntReference()
	goingToGrab[actor] = false

	recomputePath[actor] = -1.0

	sentrySafe[actor] = -1.0
	sentryCooldown[actor] = -1.0

	dispenserSafe[actor] = -1.0
	dispenserCooldown[actor] = -1.0

	findNestHint[actor] = -1.0
	advanceNestSpot[actor] = -1.0

	engine.PluginBotOf(actor).SetPathing(true)
}

/*
IsSentrySafe is whether the sentry is finished and in no trouble, asked of the
sentry rather than of a clock.

m_ctSentrySafe is this answer with three seconds of life on it, and the three
seconds are there so that the flag survives a frame where the sentry is briefly not
full. It is refreshed by the idle action and only by the idle action.

That is fine for the idle action's own branches and wrong for anything the idle
action suspends into, because suspending stops its update running: the flag then
expires three seconds later whatever the sentry is actually doing. The dispenser
build read it and ended itself with "Sentry not safe" three seconds after starting,
every time, on every map. It got a dispenser up only where the walk and the
placement both fitted inside those three seconds, which is why the dispenser was
standing for 13% of a wave on Mannworks and 45% on Mannhattan.

So the question is asked of the sentry where the answer has to be current, and the
flag stays for what it was for.
*/
//
//sp:name IsSentrySafe
func IsSentrySafe(sentry int32) bool {
	if sentry == engine.InvalidEntReference() {
		return false
	}

	if engine.IsBuildingUp(sentry) {
		return false
	}

	if engine.EntityHealth(sentry) < engine.EntityMaxHealth(sentry) {
		return false
	}

	if !engine.IsMiniBuilding(sentry) && engine.UpgradeLevel(sentry) < 3 {
		return false
	}

	return engine.EntProp(sentry, engine.PropSend(), "m_iAmmoShells") > 50
}

// ShouldAdvanceNestSpot says the front has moved past the ground he holds.
//
//sp:name CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot
func ShouldAdvanceNestSpot(actor int32) bool {
	if engine.NestAreaOf(actor) == engine.NullArea() {
		return false
	}

	obj := engine.ObjectOfType(actor, engine.ObjectSentry())

	/* Nothing to advance, and saying otherwise stopped him building at all

	The idle action asks this first and returns without doing anything when the answer is yes,
	because advancing is the thing it is about to do. An engineer whose sentry has just been
	destroyed has nothing to move, so the answer has to be no: it was yes, and he stood in the
	middle of Bigrock for the rest of the wave with no sentry, no dispenser, and every branch that
	would have built one behind that return.

	It was expensive as well as wrong. Two frames of that is two calls to PickBuildArea, and
	PickBuildArea calls GetBombInfo, which walks every nav area on the map. Sixty-six times a
	second, twice, per engineer, for as long as he had no sentry. */
	if obj == engine.InvalidEntReference() {
		return false
	}

	if advanceNestSpot[actor] <= 0.0 {
		advanceNestSpot[actor] = engine.GameTime() + 5.0
		return false
	}

	if engine.EntityHealth(obj) < engine.EntityMaxHealth(obj) {
		advanceNestSpot[actor] = engine.GameTime() + 5.0
		return false
	}

	// IsElapsed
	if engine.GameTime() > advanceNestSpot[actor] {
		advanceNestSpot[actor] = -1.0
	}

	found, bombinfo := engine.GetBombInfo()

	if !found {
		return false
	}

	flBombTargetDistance := engine.TravelDistanceToBombTarget(engine.NavArea(engine.NestAreaOf(actor)))

	// No point in advancing now.
	if flBombTargetDistance <= 1000.0 {
		return false
	}

	bigger := flBombTargetDistance > bombinfo.MaxBattleFront

	return bigger
}

/*
Deciding this costs a full nav mesh sweep per engineer

GetBombInfo walks every area, PickBuildArea walks every area and scores what
survives, and the sight score traces from each candidate to every sampled approach
area. Doing that for six engineers inside the wave_complete frame is how the
server's watchdog gets tripped: it does not care that the work is finite, only that
the frame took seconds.

So one engineer per timer tick. The shopping period is a minute and the queue is at
most the server's player count, which is a rounding error against that.
*/
//
//sp:name NEST_RELOCATE_EVAL_INTERVAL
const relocateEvalInterval = 0.1

var (
	//sp:name m_iNestRelocateEvalNext
	relocateEvalNext int32
	//sp:name m_hNestRelocateEvalTimer
	relocateEvalTimer engine.Timer
)

/*
OnWaveComplete asks every engineer whether the ground he holds is still the ground
he wants, once per wave.

Wave complete rather than wave start, because that is the last moment the old
buildings are still his to decide about: the upgrade session that follows tears
them down, and the answer here is what tells it whether to. It also gives him the
whole shopping period to act on the answer instead of carrying a sentry while the
robots walk in.
*/
//
//sp:name EngineerNestRelocation_OnWaveComplete
func OnWaveComplete() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		engine.SetNestRelocate(i, engine.NullArea())
	}

	StopEvaluating()

	if !engine.NestRelocate().Bool() {
		return
	}

	relocateEvalNext = 1
	relocateEvalTimer = engine.CreateTimer(relocateEvalInterval, EvaluateNestRelocation, engine.Default(), engine.TimerRepeat())
}

// StopEvaluating is the wave having started, or the round: whatever the queue had
// left is about a bomb that has moved.
//
//sp:name EngineerNestRelocation_StopEvaluating
func StopEvaluating() {
	StopEval()
}

// StopEval kills the timer and empties the queue.
//
//sp:name StopNestRelocateEval
func StopEval() {
	relocateEvalNext = 1

	if relocateEvalTimer != engine.NoTimer() {
		engine.KillTimer(relocateEvalTimer)
		relocateEvalTimer = engine.NoTimer()
	}
}

// EvaluateNestRelocation asks one engineer per tick.
//
//sp:name Timer_EvaluateNestRelocation
//sp:public
//nolint:revive // unused-parameter: the signature is the timer's, not ours
func EvaluateNestRelocation(timer engine.Timer) engine.Outcome {
	if relocateEvalNext > engine.MaxClients() {
		relocateEvalTimer = engine.NoTimer()
		return engine.PluginStop()
	}

	client := relocateEvalNext
	relocateEvalNext++

	if !ShouldEvaluateNestRelocation(client) {
		return engine.PluginContinue()
	}

	move, destination := engine.ShouldRelocateNest(client)

	if !move {
		return engine.PluginContinue()
	}

	engine.SetNestRelocate(client, destination)

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("EngineerNestRelocation: %N is moving nest", client)
	}

	return engine.PluginContinue()
}

// ShouldEvaluateNestRelocation skips an engineer with nothing to decide about.
//
//sp:name ShouldEvaluateNestRelocation
func ShouldEvaluateNestRelocation(client int32) bool {
	if !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) || !engine.IsPlayerAlive(client) {
		return false
	}

	if engine.PlayerClass(client) != engine.ClassEngineer() {
		return false
	}

	if engine.IsCarryingObject(client) {
		return false
	}

	sentry := engine.ObjectOfType(client, engine.ObjectSentry())

	//nolint:staticcheck // QF1001: the shipped file writes the negation this way round
	return !(sentry != engine.InvalidEntReference() && engine.IsBuildingUp(sentry))
}

// ResetAll forgets every pending relocation.
//
//sp:name EngineerNestRelocation_ResetAll
func ResetAll() {
	StopEval()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		engine.SetNestRelocate(i, engine.NullArea())
		nestBeforeRelocate[i] = engine.NullArea()
		relocateDeadline[i] = -1.0
	}
}

/*
DumpNest is what each engineer has actually got standing, and where.

An engineer who never finishes a building looks the same from outside as one who
never started, and a teleporter half of a pair looks the same as none. This says
which, and where each piece ended up, so a spot that refuses everything can be
walked to with sm_dump_spot.
*/
//
//sp:name Command_DumpNest
//sp:public
//
//nolint:revive // unused-parameter: the signature is SourceMod's, not ours
func DumpNest(client int32, args int32) engine.Outcome {
	/* How many areas a nest decision walks, which is the size of everything else here

	PickBuildArea and GetBombInfo both walk the whole mesh, so this number is the unit that any
	"why did the frame take that long" answer is counted in. */
	engine.ReplyToCommand(client, "%d nav areas on this map", engine.NavAreaCount())

	/* Every building standing, and who the game says owns it

	Asked for because a play-test found two dispensers with one engineer on the team, which is a
	thing the game does not let a player do: an engineer placing a second one has his first taken
	down for him. So one of them belongs to somebody else, or to nobody, and the per-engineer
	listing below cannot show either. This walks the entities instead of the players. */
	building := int32(-1)
	standing := int32(0)

	for {
		building = engine.FindEntityByClassname(building, "obj_*")

		if building == -1 {
			break
		}

		class := engine.EntityClassname(building)

		owner := engine.BuilderOf(building, engine.PropSend(), "m_hBuilder")
		at := engine.AbsOriginOf(building)

		var whose engine.Text

		if owner > 0 && owner <= engine.MaxClients() && engine.IsClientInGame(owner) {
			whose = engine.Format("%N", owner)
		} else {
			whose = engine.Format("nobody (orphan, owner index %d)", owner)
		}

		engine.ReplyToCommand(client, "%s #%d at %.0f %.0f %.0f, built by %s", class, building, at[0], at[1], at[2], whose)

		standing++
	}

	engine.ReplyToCommand(client, "%d buildings standing", standing)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || engine.PlayerClass(i) != engine.ClassEngineer() {
			continue
		}

		nest := engine.NestBuildPosition(engine.NestAreaOf(i))

		engine.ReplyToCommand(client, "%N: nest %.0f %.0f %.0f", i, nest[0], nest[1], nest[2])

		DumpBuilding(client, "sentry", engine.ObjectOfType(i, engine.ObjectSentry()))
		DumpBuilding(client, "dispenser", engine.ObjectOfType(i, engine.ObjectDispenser()))
		DumpBuilding(client, "entrance", engine.ObjectOfTypeMode(i, engine.ObjectTeleporter(), engine.ModeEntrance()))
		DumpBuilding(client, "exit", engine.ObjectOfTypeMode(i, engine.ObjectTeleporter(), engine.ModeExit()))

		// Asking moves the engineer's pending teleporter target, which the idle action recomputes anyway
		wants := engine.ShouldBuildTeleporter(i)

		var lastResult engine.Text

		engine.TeleporterLastResult(i, lastResult, 64)

		engine.ReplyToCommand(client, "  teleporter: round %d, sentry safe %s, gave up %s, wants %s%s, last \"%s\"",
			engine.RoundState(),
			engine.Choose(sentrySafe[i] > engine.GameTime(), "yes", "no"),
			engine.Choose(engine.TeleporterHasGivenUp(i), "yes", "no"),
			engine.Choose(wants, "yes", "no"),
			engine.Choose(engine.LookupEntityActionByName(i, "DefenderBuildTeleporter") != engine.InvalidAction(), ", building one now", ""),
			lastResult)

		if wants {
			spot := engine.TeleporterSpot(i)

			engine.ReplyToCommand(client, "  teleporter target: mode %d at %.0f %.0f %.0f",
				engine.TeleporterMode(i), spot[0], spot[1], spot[2])
		}
	}

	return engine.PluginHandled()
}

// DumpBuilding is one building, or that there is none.
//
//sp:name DumpBuilding
func DumpBuilding(client int32, what string, building int32) {
	if building == engine.InvalidEntReference() {
		engine.ReplyToCommand(client, "  %s: none", what)

		return
	}

	origin := engine.AbsOriginOf(building)

	engine.ReplyToCommand(client, "  %s: level %d, %d of %d health, at %.0f %.0f %.0f%s",
		what, engine.UpgradeLevel(building), engine.EntityHealth(building),
		engine.EntityMaxHealth(building), origin[0], origin[1], origin[2],
		engine.Choose(engine.IsBuildingUp(building), ", still going up", ""))
}
