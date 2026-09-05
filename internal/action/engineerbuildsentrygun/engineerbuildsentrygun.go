/*
Package engineerbuildsentrygun is
source/redbots3/behavior/engineerbuildsentrygun.sp.

The sentry, which is the whole of the engineer's job and was the last building
still guessing.

The dispenser and the teleporter both learned the same lesson and this had not: a
building goes down in front of the man and never under him, so walking onto the
spot and pressing fire aims the sentry at whatever is a build's reach beyond it.
The old code walked to the nest point, stood on it, and aimed at its own feet.
Between rounds that mostly worked, because the engineer is teleported onto the
point and the ground under a nest hint is usually clear; in the middle of a wave,
having walked there, it did not.

There was also no clock on any of it. No reach deadline, no give-up: an engineer
who could not place a sentry stayed in this action for the rest of the wave,
which is what a test-bed run of Bigrock's first wave looked like from outside.
Eight minutes, no sentry, and nothing in the logs saying why. Everything here has
a limit now, and running out of one hands the engineer back to the idle action,
which tries again three seconds later with a freshly scored nest.

//sp:action DefenderBuildSentrygun CTFBotMvMEngineerBuildSentrygun
*/
package engineerbuildsentrygun

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/nestsetup"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// A build's reach short of the spot, with the spot in front of him, same as the
// other two.
//
//sp:name SENTRY_BUILD_REACH
const buildReach = 90.0

/*
Eight looks at the spot, one from each side, before the spot itself is the thing
in question.

A sentry refused from one side is usually a sentry with a wall behind it rather
than a sentry on bad ground, and the answer to that is to stand somewhere else.
Re-scoring the nest on the first refusal, which is what this did, threw away a
good spot for a bad reason and cost a full pass over the nav mesh every time it
happened.
*/
//
//sp:name SENTRY_TRY_POINTS
const tryPoints = 8

// Long enough for the game to act on a press, short enough to retry one it
// refused.
//
//sp:name SENTRY_PRESS_SETTLE
const pressSettle = 0.3

//sp:name SENTRY_TRY_TIME
const tryTime = 1.5

/*
How long the walk and the whole business may take.

The walk is priced by its length, because "the walk is inside the nest" stopped
being true the moment he started every one of them at the upgrade station. Past
the build time he goes back to the idle action rather than settling for where he
stands: a sentry is not a dispenser, and one pointed at a wall is worse than
three more seconds spent finding somewhere it can see from.

The settle range is the important one. Running out of clock used to mean building
beside himself wherever he had got to, with no distance test of any kind: that is
a sentry at a random place on the map, reported from play on Coaltown, and this
file's own comment admits to one 625 units from its nest on Decoy. Two build
reaches is close enough that what he settles for still sees what the nest was
chosen to see. Further out he keeps walking, and the give-up clock hands him back
to the idle action, which scores a nest again and tries afresh.
*/
const (
	//sp:name SENTRY_REACH_TIME
	reachTime = 12.0
	//sp:name SENTRY_SETTLE_RANGE
	settleRange = 200.0
	//sp:name SENTRY_BUILD_TIME
	buildTime = 45.0
)

//sp:name m_ctSentryReachDeadline
var reachDeadline [slots.Count]float32

/*
What the stuck watchdog had counted when this build attempt started.

The watchdog resets the whole behaviour every STUCK_TIME, which restarts this
action, which re-arms the reach deadline. So an engineer who cannot reach his
spot is rescued by nothing: the deadline that would have made him pick another
spot is pushed forward every time he is rescued.

Measured on Mannworks with Mean Machines: the same engineer stuck at
1014 885 274, nineteen times, inside DefenderBuildSentrygun, and never a sentry.
STUCK_TIME and SENTRY_REACH_TIME are both twelve seconds, so the two timers chase
each other.

Counting stucks across the restarts is what survives them.
*/
//
//sp:name m_iSentryStuckMark
var stuckMark [slots.Count]int32

/*
The nest the mark above belongs to, so a restart does not look like a new
attempt.

The shipped declaration fills this with NULL_AREA explicitly. NULL_AREA is zero
and a SourcePawn global starts at zero, so the fill says nothing the declaration
does not; the generated one leaves it out.
*/
//
//sp:name m_aSentryStuckArea
var stuckArea [slots.Count]engine.Area

// Stucks inside one attempt before the spot is the suspect rather than the walk.
//
//sp:name SENTRY_STUCK_GIVEUP
const stuckGiveUp = 2

var (
	//sp:name m_ctSentryGiveUpTime
	giveUpTime [slots.Count]float32
	//sp:name m_ctSentryTryDeadline
	tryDeadline [slots.Count]float32
	// When the last build press is allowed to have landed, so the next frame
	// is not another press.
	//
	//sp:name m_ctSentryPressed
	pressed [slots.Count]float32
	//sp:name m_iSentryTry
	tryIndex [slots.Count]int32
	//sp:name m_vSentrySpot
	sentrySpot [slots.Count][3]float32
	//sp:name m_vSentryStand
	sentryStand [slots.Count][3]float32
)

// OnStart arms every clock, teleports him onto the nest between rounds, and
// marks where the stuck count stood.
func OnStart(actor int32) engine.Outcome {
	engine.UpdateLookAroundForEnemies(actor, true)

	giveUpTime[actor] = engine.GameTime() + buildTime
	tryDeadline[actor] = engine.GameTime() + tryTime
	pressed[actor] = 0.0
	tryIndex[actor] = 0

	if engine.RoundState() == engine.RoundStateBetweenRounds() {
		if engine.NestAreaOf(actor) != engine.NullArea() {
			// Teleport ourselves to the nest area for a faster setup
			vNestPosition := engine.NestBuildPosition(engine.NestAreaOf(actor))
			vNestPosition[2] += engine.StepHeight()
			engine.EntityOf(actor).SetAbsOrigin(vNestPosition)

			// The nest is the first claim of the break, and the one the other three are placed around
			if engine.Feature(engine.FeatureEngineerSetupPhase()) {
				nestsetup.ClaimSetupSpot(actor, nestsetup.SetupSentry, vNestPosition)
			}
		}
	}

	StandPoint(actor)

	// After the teleport above, so a between-rounds walk is priced from where he actually starts it
	reachDeadline[actor] = engine.GameTime() + engine.BuildReachTime(engine.AbsOriginOf(actor), sentryStand[actor])

	engine.LogBuildFailure(actor, "sentry", "started")

	/* The mark has to survive the watchdog's reset, which is the whole point of it

	ResetIntentionInterface restarts this action, so anything armed in OnStart is armed again every
	twelve seconds and can never expire. That is the fault in the reach deadline, and re-marking
	here would inherit it: the count would restart alongside the thing it counts.

	So the mark belongs to the nest rather than to the attempt. It resets when he is sent somewhere
	new, and not when he is merely restarted at the same place. */
	if stuckArea[actor] != engine.NestAreaOf(actor) {
		stuckArea[actor] = engine.NestAreaOf(actor)
		stuckMark[actor] = engine.StuckCountOf(actor)

		engine.LogBuildFailure(actor, "sentry", "new nest, stuck mark reset")
	}

	return engine.Continue()
}

// Update walks to the stand point, gets the toolbox out, and presses once per
// settle until the game accepts one.
func Update(actor int32) engine.Outcome {
	if engine.NestAreaOf(actor) == engine.NullArea() {
		engine.LogBuildFailure(actor, "sentry", "no nest area")
		return engine.Done("No hint entity")
	}

	if engine.ShouldAdvanceNestSpot(actor) {
		// And you.

		engine.LogBuildFailure(actor, "sentry", "told to advance the nest")
		return engine.Done("No sentry")
	}

	// Every side of this spot refused him and the walk is not getting shorter. The idle action retries
	if engine.GameTime() > giveUpTime[actor] {
		engine.LogBuildFailure(actor, "sentry", "every side of the spot refused him")

		return engine.Done("Nowhere here will take a sentry")
	}

	spot := sentrySpot[actor]
	stand := sentryStand[actor]

	/* The walk ran out, so he builds from where he got to rather than into whatever stopped him

	And he puts it beside himself rather than pointing it at the nest he could not reach. Aiming at
	the nest from three metres short of it is the same thing; aiming at it from twenty metres short
	puts the sentry twenty metres from where anybody wanted it, facing a direction chosen by where
	he happened to get stuck. Decoy produced one 625 units from its own nest that way. */
	rangeToSpot := engine.VectorDistance(engine.AbsOriginOf(actor), sentrySpot[actor])
	outOfTime := engine.GameTime() > reachDeadline[actor] && rangeToSpot < settleRange

	/* The walk ran out and he is nowhere near the spot, so the spot is what to give up on

	outOfTime above deliberately refuses to build from far away, for the reason in the comment on
	it. What that leaves is the case nothing handled: an engineer who never arrives keeps walking at
	a spot he cannot reach, for the whole mission, and builds nothing at all. Reported on Mannworks
	with Mean Machines, and Bigrock has a spot on a rock he cannot jump onto.

	The retry below re-scores the nest, and it only runs once he is close enough to try building. So
	the same thing is done here, from the other side of the range check: a new area rather than a
	sentry twenty metres from where anybody wanted one. */
	if (engine.GameTime() > reachDeadline[actor] && rangeToSpot >= settleRange) ||
		engine.StuckCountOf(actor)-stuckMark[actor] >= stuckGiveUp {
		stuckMark[actor] = engine.StuckCountOf(actor)
		stuckArea[actor] = engine.NestAreaOf(actor)

		engine.SetNestArea(actor, engine.PickBuildArea(actor))
		tryIndex[actor] = 0
		StandPoint(actor)
		reachDeadline[actor] = engine.GameTime() + reachTime

		engine.LogBuildFailure(actor, "sentry", "could not reach the spot, took another")

		return engine.Continue()
	}

	if outOfTime {
		stand = engine.AbsOriginOf(actor)

		_, spot = engine.BuildStandPoint(stand, sentrySpot[actor], tryIndex[actor],
			tryPoints, buildReach)
	}

	rangeToStand := engine.VectorDistance(engine.AbsOriginOf(actor), stand)
	myWeapon := engine.ActiveWeapon(actor)
	myNextbot := engine.NextBotOf(actor)
	myBody := myNextbot.Body()
	myLoco := myNextbot.Locomotion()

	if rangeToStand < 200.0 {
		// Start building a sentry
		if !engine.IsBuilderSetTo(actor, engine.ObjectSentry()) {
			engine.FakeClientCommandThrottled(actor, "build 2")
		}

		engine.UpdateLookAroundForEnemies(actor, false)

		if !myLoco.IsStuck() {
			engine.ExtraButtonsOf(actor).PressButtons(engine.InDuck(), 0.1)
		}

		// It goes where he looks, so he looks at the spot rather than at the ground under himself
		engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.1, engine.NoAddress(), "Placing sentry")
	}

	if rangeToStand > 70.0 {
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
		tryDeadline[actor] = engine.GameTime() + tryTime

		engine.PluginBotOf(actor).SetPathGoalVector(stand)
		engine.PluginBotOf(actor).SetPathing(true)

		if rangeToStand > 300.0 {
			// Fuck em up.
			engine.EquipWeaponSlot(actor, engine.WeaponSlotPrimary())
		}

		engine.UpdateLookAroundForEnemies(actor, true)

		return engine.Continue()
	}

	engine.PluginBotOf(actor).SetPathing(false)

	if myWeapon != -1 && engine.WeaponID(myWeapon) == engine.WeaponBuilder() {
		objBeingBuilt := engine.EntPropEnt(myWeapon, engine.PropSend(), "m_hObjectBeingBuilt")

		if objBeingBuilt == -1 {
			return engine.Continue()
		}

		/* One press, then a tick for the game to act on it

		The check at the end of this function runs in the same frame as this press, so it asks
		whether a sentry exists before the game has put one down. It answered no, the action
		carried on, and the toolbox re-armed: another press, another building. Measured on the
		dispenser, which has the same shape and which the test-bed caught standing twice under one
		engineer. */
		if engine.GameTime() >= pressed[actor] {
			pressed[actor] = engine.GameTime() + pressSettle

			engine.PressFireButton(actor)
		}

		/* The game says no from here, so try looking at it from the next side round

		Only once he is actually looking at it: the answer while his head is still coming round is
		the answer for wherever it was pointing, which is not this spot. */
		if !engine.IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget() &&
			engine.GameTime() > tryDeadline[actor] {
			tryIndex[actor]++

			/* Every side refused him, so now the spot itself is the thing in question

			This is where the nest gets re-scored, and not before: a pass over the nav mesh is the
			expensive answer and it was being given to a wall behind the man. */
			if tryIndex[actor] >= tryPoints {
				engine.SetNestArea(actor, engine.PickBuildArea(actor))
				tryIndex[actor] = 0
			}

			StandPoint(actor)

			tryDeadline[actor] = engine.GameTime() + tryTime
			reachDeadline[actor] = engine.GameTime() + reachTime

			return engine.Continue()
		}
	}

	sentry := engine.ObjectOfType(actor, engine.ObjectSentry())

	if sentry == engine.InvalidEntReference() {
		return engine.Continue()
	}

	engine.SetPlayerReady(actor, true)

	engine.LogBuildFailure(actor, "sentry", "built one")

	return engine.Done("Built a sentry")
}

/*
StandPoint is where the sentry goes and where he stands to put it there, on a
side he can stand on.

Sides with nothing walkable under them are skipped rather than walked at: a nest
on raised ground has thin air around it, and pathing at a coordinate in mid-air
puts the engineer on the floor below holding the toolbox until a clock saves him.
Bounded by the number of sides there are.
*/
//
//sp:name SentryStandPoint
func StandPoint(actor int32) {
	sentrySpot[actor] = engine.NestBuildPosition(engine.NestAreaOf(actor))

	for skipped := int32(0); skipped < tryPoints; skipped++ {
		ok, stand := engine.BuildStandPoint(sentrySpot[actor], engine.AbsOriginOf(actor), tryIndex[actor],
			tryPoints, buildReach)

		sentryStand[actor] = stand

		if ok {
			return
		}

		tryIndex[actor] = (tryIndex[actor] + 1) % tryPoints
	}
}

// OnEnd stops the walking and says what the attempt left behind.
func OnEnd(actor int32) {
	engine.PluginBotOf(actor).SetPathing(false)

	engine.UpdateLookAroundForEnemies(actor, true)

	/* Every way out of this action, including the ones nobody wrote a branch for

	The Done branches above name why they gave up, and a session produced far more starts than
	endings that said anything. Asking the result for its reason here printed nothing at all, which
	is what a thrown native looks like from the outside: it takes the callback with it. So this says
	only what is certainly true, which is that the attempt is over and whether it left a sentry. */
	engine.LogBuildFailure(actor, "sentry",
		engine.Choose(engine.ObjectOfType(actor, engine.ObjectSentry()) != engine.InvalidEntReference(),
			"ended with a sentry", "ended with nothing"))
}
