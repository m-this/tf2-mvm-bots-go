/*
Package engineerbuilddispenser is
source/redbots3/behavior/engineerbuilddispenser.sp.

# Which nest a named dispenser spot belongs to

A dispenser is the team's, not the sentry's. It heals and reloads whoever stands
on it, so where somebody walked the map and wrote a spot down, that is the ground
they meant, however far it sits from the nest it serves.

This kept arguing with that. First a distance bound, which a sweep of every map
killed: Bigrock's authored spots sit four to six hundred units from its authored
nests on purpose, and a bound tight enough to reject Coaltown's rejected all of
Bigrock's. Then an ownership rule and a height test, and between them they threw
away most of the authoring in the directory. Coaltown's right building ended up
with no spot at all and put its dispenser on the roof beside the sentry, which is
exactly what somebody had walked the map to avoid.

So the authored spot is respected. What chooses between several of them is the
zone where the map names one and the nearest otherwise, and the only things that
refuse a spot now are another engineer already standing a dispenser on it, and the
engineer being unable to walk there.

//sp:action DefenderBuildDispenser CTFBotMvMEngineerBuildDispenser
*/
package engineerbuilddispenser

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

//sp:name DISPENSER_SPOT_TAKEN_RANGE
const spotTakenRange = 150.0

/*
How far from the nest "where he stands" is still somewhere worth putting a dispenser

The deadline below assumes the walk is inside the nest, which it is when the
engineer is at his nest, and it no longer always is: he goes to the far end of the
map for a teleporter entrance now. A test-bed run on Coaltown found the dispenser
three thousand four hundred units from the nest, beside the spawn door, because he
lost it while he was out there and the twelve seconds ran out before he had walked
a quarter of the way back.

A dispenser two metres from the intended spot is worth all of one that never gets
built. One at the other end of the map is worth nothing at all: it feeds no sentry,
it heals nobody who is fighting, and it is a hundred metal the nest wanted. Past
this he keeps walking, and the build time below is what stops him.
*/
//
//sp:name DISPENSER_SETTLE_RANGE
const settleRange = 200.0

/*
Where he stands to put a dispenser on the spot, which is not the spot

A building goes down in front of the man, never under him. Walking onto the
coordinate and pressing fire aims the dispenser at whatever is a build's reach
beyond it, which on Coaltown is the wall on the right: the placement never comes up
green, and the engineer stands on the spot holding the toolbox until the wave starts
without him.

So he stops a reach short of the spot and looks at it. The old code turned him on
the spot instead, a tenth of a second of IN_RIGHT at a time, which cannot help: the
direction he faces is the direction the dispenser goes, so turning moves the problem
rather than solving it.

When the game still says no, he walks to the next point around the spot and looks at
it from there. Eight of them, which is a look from every side at forty five degrees.
*/
const (
	//sp:name DISPENSER_BUILD_REACH
	buildReach = 90.0
	//sp:name DISPENSER_TRY_POINTS
	tryPoints = 8
	//sp:name DISPENSER_TRY_TIME
	tryTime = 2.0
)

/*
How long one build press is given to land before another is allowed

The press puts the building down on the tick after it, so asking the game whether a
dispenser exists in the same frame asks a question it has not answered yet. It
answered "none", and the action pressed fire again: two dispensers standing, one
engineer, and the test-bed counting held:2 built:2 listed:2 eighteen times in four
waves.

Long enough for the game to act and short enough that a press the game refused is
retried while the engineer is still looking at the spot.
*/
//
//sp:name DISPENSER_PRESS_SETTLE
const pressSettle = 0.3

/*
How long he may spend on the whole business before he goes back to the wave

The readiness gate holds a wave until the engineer's nest is finished, and a nest is
not finished without a dispenser. An engineer who can never place one is an engineer
holding every wave for the length of that grace, which is what a spot with no room
around it costs.

Long enough to cover the longest walk BuildReachTime will price plus the eight looks
around the spot, because a give-up clock that expires during the walk is a give-up
clock that never lets him arrive.
*/
//
//sp:name DISPENSER_BUILD_TIME
const buildTime = 45.0

var (
	//sp:name m_ctDispenserReachDeadline
	reachDeadline [Slots]float32
	//sp:name m_ctDispenserGiveUpTime
	giveUpTime [Slots]float32
	//sp:name m_ctDispenserTryDeadline
	tryDeadline [Slots]float32
	// When the last build press is allowed to have landed, so the next frame is not another press
	//
	//sp:name m_ctDispenserPressed
	pressed [Slots]float32
	//sp:name m_iDispenserTry
	tryIndex [Slots]int32
	//sp:name m_vDispenserSpot
	dispenserSpot [Slots][3]float32
	//sp:name m_vDispenserStand
	dispenserStand [Slots][3]float32
)

// OnStart picks the spot once and prices the walk to it.
func OnStart(actor int32) engine.Outcome {
	engine.UpdateLookAroundForEnemies(actor, true)

	giveUpTime[actor] = engine.GameTime() + buildTime
	tryDeadline[actor] = engine.GameTime() + tryTime
	pressed[actor] = 0.0
	tryIndex[actor] = 0

	// Once, here, because the Update runs every tick and a path computation does not belong there
	configured, spot := ConfiguredSpot(actor)

	dispenserSpot[actor] = spot

	if !configured {
		if engine.NestAreaOf(actor) != engine.NullArea() {
			dispenserSpot[actor] = engine.RandomPointIn(engine.NestAreaOf(actor))
		} else {
			dispenserSpot[actor] = engine.AbsOriginOf(actor)
		}
	}

	// Sides he cannot stand on are skipped here rather than walked at and waited out
	ok, stand := StandPoint(actor, tryIndex[actor])

	dispenserStand[actor] = stand

	if !ok {
		NextStandPoint(actor)
	}

	/* Priced by the walk, because the spot the map names is not always next to the nest

	Coaltown's right-hand spot is 857 units from the nest it serves, on purpose, and he starts the
	walk at the upgrade station. A flat twelve seconds expired somewhere along the way and he built
	it wherever that was, which is how a hand-walked spot turned into a dispenser beside the
	teleporter entrance. */
	reachDeadline[actor] = engine.GameTime() + engine.BuildReachTime(engine.AbsOriginOf(actor), dispenserStand[actor])

	return engine.Continue()
}

// Update walks to the stand point and presses once per settle until one goes
// down.
func Update(actor int32) engine.Outcome {
	if engine.NestAreaOf(actor) == engine.NullArea() {
		engine.LogBuildFailure(actor, "dispenser", "no nest area")
		return engine.Done("No hint entity")
	}

	sentry := engine.ObjectOfType(actor, engine.ObjectSentry())

	if sentry == engine.InvalidEntReference() {
		// Fuck you.

		engine.LogBuildFailure(actor, "dispenser", "no sentry to feed")
		return engine.Done("No sentry")
	}

	/* Asked of the sentry, not of the flag the idle action keeps

	Suspending the idle action stops its update running, so its three second flag expires three
	seconds after this one starts however well the sentry is doing. This ended itself on that,
	every time, and only ever finished a dispenser where the walk and the placement both fitted
	inside those three seconds. */
	if !engine.IsSentrySafe(sentry) {
		engine.LogBuildFailure(actor, "dispenser", "sentry under fire")
		return engine.Done("Sentry not safe")
	}

	if engine.ShouldAdvanceNestSpot(actor) {
		// Fuck you too.

		engine.LogBuildFailure(actor, "dispenser", "told to advance the nest")
		return engine.Done("Need to advance nest")
	}

	/* The spot is chosen once, not every frame

	Choosing it here used to mean a path computation per configured spot per tick per engineer,
	which is how the server's watchdog came to fire inside NavAreaBuildPath. A spot that was
	reachable when the action started is reachable a second later, and if it is not, the deadline
	below is what answers for it. */
	// Every side of the spot refused him, and a wave held for one dispenser is the worse trade
	if engine.GameTime() > giveUpTime[actor] {
		engine.LogBuildFailure(actor, "dispenser", "ran out of time to place it")

		return engine.Done("Nowhere to put a dispenser")
	}

	spot := dispenserSpot[actor]
	stand := dispenserStand[actor]

	/* The walk ran out of time, so he builds from where he stands and aims at the spot anyway

	Only while he is somewhere near his nest. Settling where he stands is a trade of accuracy for a
	dispenser that exists, and it stops being a trade at all once he is far enough away that what
	he settles for feeds nothing. */
	outOfTime := reachDeadline[actor] > 0.0 && engine.GameTime() > reachDeadline[actor] &&
		engine.VectorDistance(engine.AbsOriginOf(actor), spot) < settleRange

	if outOfTime {
		stand = engine.AbsOriginOf(actor)
	}

	/* He never arrived, so the spot is unreachable rather than slow

	outOfTime above settles for where he stands, and only while he is near the nest, for the reason
	in the comment on it. That leaves an engineer who never gets near at all walking at the same
	spot for the rest of the mission.

	He gives the dispenser up rather than standing there. The action ends, the idle behaviour picks
	again, and a dispenser he does not have is worth less than an engineer who is doing something
	else. */
	if reachDeadline[actor] > 0.0 && engine.GameTime() > reachDeadline[actor] &&
		engine.VectorDistance(engine.AbsOriginOf(actor), spot) >= settleRange {
		engine.LogBuildFailure(actor, "dispenser", "could not reach the spot, gave it up")

		return engine.Done("Cannot reach the dispenser spot")
	}

	rangeToStand := engine.VectorDistance(engine.AbsOriginOf(actor), stand)

	myNextbot := engine.NextBotOf(actor)
	myBody := myNextbot.Body()

	if rangeToStand < 200.0 {
		// Start building a dispenser
		if !engine.IsBuilderSetTo(actor, engine.ObjectDispenser()) {
			engine.FakeClientCommandThrottled(actor, "build 0")
		}

		// It goes where he looks, so he looks at the spot. Turning on the spot only turns the problem
		engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.1, engine.AddressNull(), "Placing dispenser")

		// NOTE: we do not look around for incoming enemies cause all we care about is placing this dispenser
	}

	if rangeToStand > 70.0 {
		engine.PluginBotOf(actor).SetPathGoalVector(stand)
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

		/* The game says no from here, so try looking at it from the next side

		Only once he is actually looking at the spot: the answer while his head is still coming
		round is the answer for wherever it was pointing, which is not this spot. */
		if !engine.IsPlacementOK(objBeingBuilt) && !outOfTime &&
			myBody.IsHeadAimingOnTarget() && engine.GameTime() > tryDeadline[actor] {
			NextStandPoint(actor)

			return engine.Continue()
		}
	}

	/* Asked before the press, not after it

	It used to press and then ask in the same frame, which is a frame too early: the answer was
	always "no dispenser", so the next tick pressed again and put a second one down. */
	dispenser := engine.ObjectOfType(actor, engine.ObjectDispenser())

	if dispenser != engine.InvalidEntReference() {
		engine.SetPlayerReady(actor, true)

		return engine.Done("Built a dispenser")
	}

	// A press already given its chance is not given another until the game has had its tick
	if engine.GameTime() < pressed[actor] {
		return engine.Continue()
	}

	pressed[actor] = engine.GameTime() + pressSettle

	engine.PressFireButton(actor)

	return engine.Continue()
}

/*
StandPoint is one build's reach short of the spot, on the side the try asks for,
and on ground he can stand on.

Try zero is the side he is walking in from, so the first look costs him no walking
at all. Each one after it is forty five degrees round from there. False when the nav
mesh has nothing walkable there, which is the caller's cue to go round to the next
one.
*/
//
//sp:name DispenserStandPoint
func StandPoint(actor int32, attempt int32) (ok bool, stand [3]float32) {
	ok, stand = engine.BuildStandPoint(dispenserSpot[actor], engine.AbsOriginOf(actor), attempt,
		tryPoints, buildReach)

	return ok, stand
}

// NextStandPoint is the next side of the spot he can actually stand on, or the
// end of them, which is when he settles.
//
//sp:name NextDispenserStandPoint
func NextStandPoint(actor int32) {
	for tryIndex[actor]++; tryIndex[actor] < tryPoints; tryIndex[actor]++ {
		ok, stand := StandPoint(actor, tryIndex[actor])

		dispenserStand[actor] = stand

		if !ok {
			continue
		}

		tryDeadline[actor] = engine.GameTime() + tryTime

		return
	}

	// A dispenser two metres from the spot beats an engineer who never builds one
	reachDeadline[actor] = engine.GameTime()
}

// OnEnd puts his eyes back on the wave.
func OnEnd(actor int32) {
	engine.UpdateLookAroundForEnemies(actor, true)
}

/*
ConfiguredSpot is the dispenser spot the map configuration asks for, false when it
asks for nothing.

Nearest to the nest rather than to the engineer, because he walks back to the nest
anyway and the dispenser is there to feed the sentry.
*/
//
//sp:name ConfiguredDispenserSpot
func ConfiguredSpot(actor int32) (found bool, spot [3]float32) {
	spots := engine.DispenserSpots()

	if spots.Length() == 0 {
		return false, spot
	}

	// The authored point rather than the area centre, so the comparison is like with like
	nest := engine.NestBuildPosition(engine.NestAreaOf(actor))

	/* The zone this nest belongs to, when the map names one, decides before distance does

	Coaltown is why. The ground behind the wall on the right is eight hundred units from the nest
	it serves and two hundred from a different one, so nearest is simply the wrong answer there and
	no distance rule was ever going to fix it: the map has to be able to say which spot goes with
	which nest, and a zone is how it already says that about nests. */
	myZone := engine.NestZoneOf(engine.NestAreaOf(actor))

	/* His own zone if the map put a spot in it, and the spots belonging to nobody if it did not

	Two passes rather than one condition, because the two ideas are different. A spot in a zone is
	reserved for it: Coaltown's right building has one and nothing else may take it, or the nearest
	rule hands it to the nest in the middle. A nest in a zone is a separate and older idea, about
	spreading engineers over the map, and it must not stop that engineer using a spot nobody
	reserved. Mannhattan names zones on all four of its nests and on none of its spots, and one
	condition covering both left it with nothing at all. */
	free := engine.NewBlocks(3)
	defer free.Close()

	/* A spot the path query refused is the last resort, not a spot that stopped existing

	The query is the same ComputeToPos that was measured returning nothing at all for a medic with
	a live patient in front of him, and it is asked here from wherever the engineer happens to be
	standing. Before the first wave that is his nest, and it answers; between later waves it is the
	upgrade station at the other end of the map, and when it refuses, the coordinate somebody
	walked the map to find is silently dropped and he builds wherever the fallback puts him.

	Reported exactly that way: right before wave one, wrong from then on. So an unreachable
	authored spot is still an authored spot, and it is used when nothing better is offered. */
	refused := engine.NewBlocks(3)
	defer refused.Close()

	CollectSpots(actor, myZone, free, refused)

	if free.Length() == 0 && myZone[0] != 0 {
		CollectSpots(actor, engine.NoZone(), free, refused)
	}

	found, spot = engine.NearestConfiguredSpot(free, nest)

	if !found {
		found, spot = engine.NearestConfiguredSpot(refused, nest)
	}

	if engine.ManagerDebug().Bool() {
		if found {
			engine.PrintToServer("ConfiguredDispenserSpot: %N takes the named spot %.0f %.0f %.0f", actor, spot[0], spot[1], spot[2])
		} else {
			engine.PrintToServer("ConfiguredDispenserSpot: %N has no named spot for the nest at %.0f %.0f %.0f", actor, nest[0], nest[1], nest[2])
		}
	}

	return found, spot
}

// CollectSpots is every named spot in one zone nobody has taken, split by whether
// the mesh will admit a path today.
//
//sp:name CollectDispenserSpots
func CollectSpots(actor int32, wanted engine.Text, free engine.List, refused engine.List) {
	spots := engine.DispenserSpots()
	zones := engine.DispenserZones()

	for i := int32(0); i < spots.Length(); i++ {
		var zone engine.Text

		if i < zones.Length() {
			zone = zones.GetString(i)
		}

		if !engine.StrEqualText(zone, wanted) {
			continue
		}

		candidate := spots.GetArray(i)

		// Somebody else's dispenser standing on it is the one thing that does rule a spot out
		if IsSpotTaken(actor, candidate) {
			continue
		}

		if engine.IsPathToVectorPossible(actor, candidate) {
			free.PushArray(candidate)
		} else {
			refused.PushArray(candidate)
		}
	}
}

// IsSpotTaken spreads several engineers over the spots the map names instead of
// stacking them on the nearest one.
//
//sp:name IsDispenserSpotTaken
func IsSpotTaken(actor int32, spot [3]float32) bool {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor || !engine.IsClientInGame(i) {
			continue
		}

		dispenser := engine.ObjectOfType(i, engine.ObjectDispenser())

		if dispenser == engine.InvalidEntReference() {
			continue
		}

		if engine.VectorDistance(spot, engine.AbsOriginOf(dispenser)) < spotTakenRange {
			return true
		}
	}

	return false
}
