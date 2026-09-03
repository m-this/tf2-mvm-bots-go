/*
Package engineerbuilddisposable is
source/redbots3/behavior/engineerbuilddisposable.sp.

The disposable sentry, put somewhere on purpose.

Nothing placed one. The upgrade was bought, the game handed the engineer a
second sentry the next time he pressed build, and it went down wherever he
happened to be facing: reported from play as minis pointing at walls and wedged
into corners. GetObjectOfType skips disposable buildings entirely, so the rest of
the mod could not even see that it had happened.

It goes beside the real one now, because that is what it is for: the nest spot
was chosen for what it can see, and a second gun three metres from it sees the
same ground. The ring is walked the same way the teleporter exit walks the nest,
and a point with no line to the bomb is skipped rather than built on, which is
the difference between a second gun and a second thing for a giant to break.

Between rounds only. A disposable sentry is a hundred metal and a walk, and doing
either in the middle of a wave is the engineer not repairing the sentry that
matters.

//sp:action DefenderBuildDisposable CTFBotMvMEngineerBuildDisposable
*/
package engineerbuilddisposable

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// Far enough not to be inside the real one, near enough to hold the same ground.
const (
	//sp:name DISPOSABLE_RING_RADIUS
	ringRadius = 170.0
	//sp:name DISPOSABLE_BUILD_REACH
	buildReach = 90.0
	//sp:name DISPOSABLE_TRY_POINTS
	tryPoints = 8
	//sp:name DISPOSABLE_TRY_TIME
	tryTime = 1.5
)

// Long enough to walk round the nest and try every side of it, and no longer.
//
//sp:name DISPOSABLE_BUILD_TIME
const buildTime = 20.0

var (
	//sp:name m_ctDisposableGiveUp
	giveUp [slots.Count]float32
	//sp:name m_ctDisposableTryDeadline
	tryDeadline [slots.Count]float32
	//sp:name m_iDisposableTry
	tryIndex [slots.Count]int32
	//sp:name m_vDisposableSpot
	spot [slots.Count][3]float32
	//sp:name m_vDisposableStand
	stand [slots.Count][3]float32
	//sp:name m_bDisposableGaveUp
	gaveUp [slots.Count]bool
)

// OnStart starts the clock and picks the first side of the nest to try.
func OnStart(actor int32) engine.Outcome {
	giveUp[actor] = engine.GameTime() + buildTime
	tryDeadline[actor] = engine.GameTime() + tryTime
	tryIndex[actor] = 0

	StandPoint(actor)

	engine.UpdateLookAroundForEnemies(actor, true)

	return engine.Continue()
}

// Update walks to the spot, gets the toolbox out on the way, and tries the next
// side of the nest when the game refuses this one.
func Update(actor int32) engine.Outcome {
	// The wave is what the real sentry is for, and this is not worth a second of it
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return engine.Done("Wave started")
	}

	sentry := engine.ObjectOfType(actor, engine.ObjectSentry())

	if sentry == engine.InvalidEntReference() {
		return engine.Done("No sentry to stand one beside")
	}

	if CountDisposableSentries(actor) >= DisposableSentriesAllowed(actor) {
		engine.PluginBotOf(actor).SetPathing(false)

		return engine.Done("Built one")
	}

	if giveUp[actor] < engine.GameTime() {
		gaveUp[actor] = true

		return engine.Done("Nowhere beside the nest will take one")
	}

	mySpot := spot[actor]
	myStand := stand[actor]

	buildRange := engine.VectorDistance(engine.AbsOriginOf(actor), myStand)

	myNextbot := engine.NextBotOf(actor)
	myBody := myNextbot.Body()

	// The toolbox comes out on the way in, so arriving is not another wait
	if buildRange < 200.0 {
		if !engine.IsBuilderSetTo(actor, engine.ObjectSentry()) {
			engine.FakeClientCommandThrottled(actor, "build 2")
		}

		// It goes where he looks, so he looks at the spot rather than at his own feet
		engine.AimHeadTowards(myBody, mySpot, engine.AimMandatory(), 0.1, engine.NoAddress(), "Placing disposable sentry")
	}

	if buildRange > 70.0 {
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
		tryDeadline[actor] = engine.GameTime() + tryTime

		engine.PluginBotOf(actor).SetPathGoalVector(myStand)
		engine.PluginBotOf(actor).SetPathing(true)

		return engine.Continue()
	}

	engine.PluginBotOf(actor).SetPathing(false)

	myWeapon := engine.ActiveWeapon(actor)

	if myWeapon != -1 && engine.WeaponID(myWeapon) == engine.WeaponBuilder() {
		objBeingBuilt := engine.EntPropEnt(myWeapon, engine.PropSend(), "m_hObjectBeingBuilt")

		// The toolbox is out but the game has not decided yet
		if objBeingBuilt == -1 {
			return engine.Continue()
		}

		if !engine.IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget() &&
			engine.GameTime() > tryDeadline[actor] {
			tryIndex[actor]++

			if tryIndex[actor] >= tryPoints {
				gaveUp[actor] = true

				return engine.Done("Every side of the nest refused one")
			}

			StandPoint(actor)

			return engine.Continue()
		}
	}

	engine.PressFireButton(actor)

	return engine.Continue()
}

/*
StandPoint is where this attempt puts it, and where he stands to put it there.

Round the sentry rather than round the nest centre, because the sentry is the
thing it is meant to stand beside, and he stands between the two so the gun goes
down in front of him.
*/
//
//sp:name DisposableStandPoint
func StandPoint(actor int32) {
	sentry := engine.ObjectOfType(actor, engine.ObjectSentry())

	if sentry == engine.InvalidEntReference() {
		return
	}

	at := engine.AbsOriginOf(sentry)

	_, spot[actor] = engine.BuildStandPoint(at, engine.AbsOriginOf(actor), tryIndex[actor],
		tryPoints, ringRadius)

	_, stand[actor] = engine.BuildStandPoint(at, engine.AbsOriginOf(actor), tryIndex[actor],
		tryPoints, ringRadius-buildReach)
}

// OnEnd stops the walking and gives the looking back.
func OnEnd(actor int32) {
	engine.PluginBotOf(actor).SetPathing(false)

	engine.UpdateLookAroundForEnemies(actor, true)
}

// ForgetGivingUp is a new wave being a new chance at ground that refused him
// last time.
//
//sp:name EngineerDisposable_ForgetGivingUp
func ForgetGivingUp() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		gaveUp[i] = false
	}
}

// DisposableSentriesAllowed is how many the upgrade he bought entitles him to,
// and none at all when he has not bought it.
//
//sp:name DisposableSentriesAllowed
func DisposableSentriesAllowed(client int32) int32 {
	return engine.AttribHookValueInt(0, "engy_disposable_sentries", client)
}

/*
CountDisposableSentries is how many he has standing, which nothing else in the
mod counts.

GetObjectOfType walks past disposable buildings on purpose: everywhere else in
this mod "the sentry" means the real one, and a mini answering that question
would have the engineer defending a nest he has not built. This is the one place
that wants the other answer.
*/
//
//sp:name CountDisposableSentries
func CountDisposableSentries(client int32) int32 {
	count := int32(0)
	objects := engine.PlayerObjectCount(client)

	for i := int32(0); i < objects; i++ {
		owned := engine.PlayerObject(client, i)

		if engine.ObjectType(owned) == engine.ObjectSentry() && engine.IsDisposableBuilding(owned) {
			count++
		}
	}

	return count
}

/*
ShouldBuildDisposable is whether he should go and stand one beside the nest.

After the nest is finished and before the wave, and only where the gun would see
the ground the real one sees: a mini behind a wall is a hundred metal and a thing
for a giant to break on its way past.
*/
//
//sp:name ShouldBuildDisposable
func ShouldBuildDisposable(actor int32) bool {
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return false
	}

	if gaveUp[actor] {
		return false
	}

	if DisposableSentriesAllowed(actor) < 1 {
		return false
	}

	if CountDisposableSentries(actor) >= DisposableSentriesAllowed(actor) {
		return false
	}

	// The nest first, always: a mini is what he does with what is left over
	if engine.ObjectOfType(actor, engine.ObjectSentry()) == engine.InvalidEntReference() {
		return false
	}

	return engine.ObjectOfType(actor, engine.ObjectDispenser()) != engine.InvalidEntReference()
}
