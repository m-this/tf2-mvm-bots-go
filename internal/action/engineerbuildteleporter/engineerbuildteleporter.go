/*
Package engineerbuildteleporter is
source/redbots3/behavior/engineerbuildteleporter.sp.

# A teleporter, but only when it costs the team nothing

An engineer is a sentry and the metal that keeps it firing. Everything here is
what he does with the time left over, so it happens between rounds, with the nest
already standing, and it is abandoned the moment the sentry needs him. A wave that
arrives to find the engineer walking back from spawn with a teleporter in his hands
has already been made worse by it.

The entrance goes on the way out of spawn, read off the nav mesh's own route rather
than guessed from spawn geometry, and the exit goes beside the nest rather than on
top of it.

EngineerTeleporter_LastResult is the one function of this file still hand-written,
in source/redbots3/teleporter_result.sp: it copies into a buffer its caller sized,
and a generated function cannot see that length. mvm-z83 carries it.

//sp:action DefenderBuildTeleporter CTFBotMvMEngineerBuildTeleporter
*/
package engineerbuildteleporter

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

/*
One half of a teleporter, walk included

The entrance is at the far end of the map from the nest, so most of this is
walking: Coaltown is about ten seconds each way at an engineer's speed, and the
attempts after that are a few seconds each. Short enough that a wave never starts
without the engineer, and the readiness grace bounds the pair of them whatever this
says.
*/
//
//sp:name TELEPORTER_BUILD_MAX_TIME
const buildMaxTime = 40.0

/*
How long he may spend walking to where the exit goes before he builds it where he stands

The same rock the dispenser found on Rottenburg. A nav mesh says ground is
connected; it does not promise a bot can squeeze past a boulder to reach the middle
of it, and the old code asked again next frame forever.

The exit only. The entrance has the length of the map to walk and no business being
dropped wherever the walk stopped, so it is bounded by the build time above and
gives up rather than settling.
*/
//
//sp:name TELEPORTER_EXIT_REACH_TIME
const exitReachTime = 12.0

/*
Getting up onto the ground the map named, instead of building at the bottom of it

Bigrock's exit spot is on a rock about seventy units above the floor beside it.
Nothing in this mod has ever pressed a jump at a piece of ground, so the walk
stopped at the foot of the rock, all eight placements were refused from down there,
and the exit went down where he stood, which is the lane every robot walks.
Reported from play twice: "wtf is this exit spot", then "IT'S STILL IN MAIN BOT
PATH".

A rise of more than a step and no more than a crouch jump, from close enough to land
on it: he looks at the spot, walks into it and crouch jumps, which is what a person
does. Bounded by a count as well as by the reach clock above, because a spot that is
not actually a ledge would otherwise be an engineer hopping at a wall until the wave
starts.
*/
const (
	//sp:name TELEPORTER_CLIMB_RISE_MIN
	climbRiseMin = 24.0
	//sp:name TELEPORTER_CLIMB_RISE_MAX
	climbRiseMax = 72.0
	//sp:name TELEPORTER_CLIMB_RANGE
	climbRange = 140.0
	//sp:name TELEPORTER_CLIMB_INTERVAL
	climbInterval = 0.7
	//sp:name TELEPORTER_CLIMB_HOLD
	climbHold = 0.3
	//sp:name TELEPORTER_CLIMB_LIMIT
	climbLimit = 6
)

// He stands a build's reach short of where it goes, because a building lands in front of the man
//
//sp:name TELEPORTER_BUILD_REACH
const buildReach = 90.0

/*
Stepping out of the spawn door until the floor takes it

An entrance belongs where a player leaving spawn walks into it, so the first attempt
is a little way out of the door and each one after it is a step further along the
route to the nest. A doorway itself never takes one, and neither does the ground a
respawning player stands on.
*/
const (
	//sp:name TELEPORTER_SPAWN_OFFSET
	spawnOffset = 200.0
	//sp:name TELEPORTER_SPAWN_STEP
	spawnStep = 150.0
)

/*
The exit stands off the nest centre rather than on it

The nest centre is where the sentry is, so eight stand points looking at the centre
are eight looks at the sentry and eight refusals. The spot walks round the nest
instead, and he stands between the two: far enough out not to be inside his own
sentry, a build's reach short of where the exit goes.
*/
//
//sp:name TELEPORTER_EXIT_RADIUS
const exitRadius = 150.0

/*
And the sentry is the thing sentry busters are sent to kill

150 is a build-validity radius: far enough out not to be inside his own sentry. It
answers where he can physically place the exit and never asks where it should be.
BUSTER_BLAST_RANGE is 400, so the exit sat at well under half the reach of the one
robot whose whole job is to detonate on the nest, and a single buster took the sentry
and the team's forward spawn together.

The safe ring is tried first, all the way round. The tight ring is still there behind
it, because a map with no room at 500 units should still get an exit rather than
none: an exit that dies with the sentry beats an engineer who gives up.
*/
//
//sp:name TELEPORTER_EXIT_RINGS
const exitRings = 2

const (
	//sp:name TELEPORTER_TRY_POINTS
	tryPoints = 8
	//sp:name TELEPORTER_TRY_TIME
	tryTime = 1.5
)

var (
	//sp:name m_ctTeleporterGiveUp
	giveUp [Slots]float32
	//sp:name m_ctTeleporterReachDeadline
	reachDeadline [Slots]float32
	//sp:name m_ctTeleporterTryDeadline
	tryDeadline [Slots]float32
	//sp:name m_ctTeleporterClimb
	climbAt [Slots]float32
	//sp:name m_iTeleporterClimbs
	climbs [Slots]int32
	//sp:name m_iTeleporterTry
	tryIndex [Slots]int32
	//sp:name m_nTeleporterMode
	mode [Slots]engine.ObjectMode
	//sp:name m_vTeleporterSpot
	spotOf [Slots][3]float32
	//sp:name m_vTeleporterStand
	standOf [Slots][3]float32
	//sp:name m_vTeleporterSpawn
	spawnOf [Slots][3]float32
	//sp:name m_vTeleporterNest
	nestOf [Slots][3]float32
	/* The way out of spawn, read once while he is still standing at the far end of it

	Read per attempt instead, it was read from wherever he had walked to, and the second attempt asked
	a two hundred and ninety unit route for a point three hundred and fifty units from spawn. He tried
	one place on Coaltown and gave up. */
	//
	//sp:name m_vTeleporterRouteSpot
	routeSpot [Slots][tryPoints][3]float32
	//sp:name m_vTeleporterRouteStand
	routeStand [Slots][tryPoints][3]float32
	//sp:name m_iTeleporterRoutePoints
	routePoints [Slots]int32
	// The map named the spot, so the attempts walk around it instead of out of the spawn door
	//
	//sp:name m_bTeleporterNamedSpot
	namedSpot [Slots]bool
	/* He tried everything and none of it worked, so he stops asking until the next wave is over

	Without this the idle action suspends into this one again the moment it ends, which is an engineer
	walking the same refused route for the rest of the round, and readiness waiting on him while he
	does it. */
	//
	//sp:name m_bTeleporterGaveUp
	gaveUp [Slots]bool
	// Why the last attempt ended, for sm_dump_nest, since every give-up looks the same from outside
	//
	//sp:name m_sTeleporterLastResult
	lastResult [Slots]engine.Text
)

// OnStart reads the route out of spawn while he is still standing at his nest.
func OnStart(actor int32) engine.Outcome {
	giveUp[actor] = engine.GameTime() + buildMaxTime
	reachDeadline[actor] = engine.GameTime() + exitReachTime
	tryDeadline[actor] = engine.GameTime() + tryTime
	climbAt[actor] = 0.0
	climbs[actor] = 0
	tryIndex[actor] = 0
	routePoints[actor] = 0

	// While he is at his nest, which is the only place the whole route can be read from
	if mode[actor] == engine.ModeEntrance() && !namedSpot[actor] {
		routePoints[actor], routeSpot[actor], routeStand[actor] = engine.SpawnRoutePoints(actor, spawnOf[actor],
			spawnOffset, spawnStep, buildReach)
	}

	if !StandPoint(actor) {
		gaveUp[actor] = true

		return Ended(engine.ThisAction(), actor, "No route out of spawn to walk")
	}

	engine.UpdateLookAroundForEnemies(actor, true)

	return engine.Continue()
}

// Update walks to the spot, climbs onto it when the map put it on a rock, and
// presses fire.
func Update(actor int32) engine.Outcome {
	// The sentry outranks this, always
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return Ended(engine.ThisAction(), actor, "Wave started")
	}

	if engine.ObjectOfType(actor, engine.ObjectSentry()) == engine.InvalidEntReference() {
		return Ended(engine.ThisAction(), actor, "No sentry to leave behind")
	}

	if engine.ObjectOfTypeMode(actor, engine.ObjectTeleporter(), mode[actor]) != engine.InvalidEntReference() {
		engine.PluginBotOf(actor).SetPathing(false)

		return Ended(engine.ThisAction(), actor, "Built one")
	}

	if giveUp[actor] < engine.GameTime() {
		gaveUp[actor] = true

		return Ended(engine.ThisAction(), actor, "Ran out of time")
	}

	/* The walk to the named exit spot ran out, so he takes the ring round his own nest

	Where he stands when a walk fails is halfway to wherever he was going, and for the exit that is
	the lane the robots come down. The nest ring is a spot rather than an accident: beside his own
	sentry, out of the buster's blast, and it is where the exit goes on every map that names none. */
	if engine.Feature(engine.FeatureEngineerClimbs()) &&
		mode[actor] == engine.ModeExit() &&
		engine.GameTime() > reachDeadline[actor] {
		FallBackToNest(actor)
	}

	spot := spotOf[actor]

	// The walk to the exit ran out and the nest ring is gone too, so it goes down where he stands
	outOfTime := mode[actor] == engine.ModeExit() &&
		engine.GameTime() > reachDeadline[actor]

	myNextbot := engine.NextBotOf(actor)
	myBody := myNextbot.Body()

	/* Say when the climb is not even asked for, so silence means one thing

	Three candidates for why the jump never lands, and the third is that this branch never runs.
	Without a line here that reads the same as the debug being off. */
	if engine.DebugActions().Bool() && engine.Feature(engine.FeatureEngineerClimbs()) &&
		(outOfTime || !namedSpot[actor]) {
		engine.PrintToServer("[teleclimb] %N not asked: out of time %d, named spot %d",
			actor, outOfTime, namedSpot[actor])
	}

	// The map put the spot on top of something, so he gets on top of it rather than building below it
	if engine.Feature(engine.FeatureEngineerClimbs()) && !outOfTime && namedSpot[actor] &&
		ClimbToSpot(actor, myBody, spot) {
		engine.PluginBotOf(actor).SetPathing(false)

		return engine.Continue()
	}

	// Read after the climb, which moves it to where he landed
	stand := standOf[actor]

	if outOfTime {
		stand = engine.AbsOriginOf(actor)
	}

	teleporterRange := engine.VectorDistance(engine.AbsOriginOf(actor), stand)

	// The toolbox comes out on the way in, so arriving is not another two seconds of standing about
	if teleporterRange < 200.0 {
		if !engine.IsBuilderSetToMode(actor, engine.ObjectTeleporter(), mode[actor]) {
			engine.FakeClientCommandThrottled(actor,
				engine.Choose(mode[actor] == engine.ModeEntrance(), "build 1 0", "build 1 1"))
		}

		// It goes where he looks, so he looks at the spot
		engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.1, engine.AddressNull(), "Placing teleporter")
	}

	if teleporterRange > 70.0 {
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
		tryDeadline[actor] = engine.GameTime() + tryTime

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

		/* This floor will not take it, so try the next place that might

		Only once he is actually looking at the spot: the answer while his head is still coming
		round is the answer for wherever it was pointing, which is not this spot. */
		if !engine.IsPlacementOK(objBeingBuilt) && !outOfTime &&
			myBody.IsHeadAimingOnTarget() && engine.GameTime() > tryDeadline[actor] {
			tryIndex[actor]++

			if tryIndex[actor] >= TryLimit(actor) || !StandPoint(actor) {
				// The exit goes down here, and an entrance nowhere near the spawn door goes nowhere
				if mode[actor] != engine.ModeExit() {
					gaveUp[actor] = true

					return Ended(engine.ThisAction(), actor, "Nowhere out of spawn takes one")
				}

				// Nothing round the named spot takes one, so the nest ring gets its own eight tries
				if !engine.Feature(engine.FeatureEngineerClimbs()) || !FallBackToNest(actor) {
					reachDeadline[actor] = engine.GameTime()
				}

				return engine.Continue()
			}

			reachDeadline[actor] = engine.GameTime() + exitReachTime

			return engine.Continue()
		}
	}

	engine.PressFireButton(actor)

	return engine.Continue()
}

/*
TryLimit is how many placements he will try before he gives up.

The exit walks two rings rather than one, so it gets two rounds of the same eight
angles. Every other case has one spot or one route and is unchanged.
*/
//
//sp:name TeleporterTryLimit
func TryLimit(actor int32) int32 {
	if !namedSpot[actor] && mode[actor] == engine.ModeExit() {
		return tryPoints * exitRings
	}

	return tryPoints
}

/*
SayClimb says why a climb was refused, or that it was tried.

Measured on Bigrock the jump never landed: no sample under 100 units from the spot,
and the minimum equal to the median. Three candidates were left and one line
separates them, because each writes a different reason here: the 24 to 72 window not
matching the real rise, the jump not carrying, or the branch never being reached at
all. See mvm-fgs.
*/
//
//sp:name SayClimb
func SayClimb(actor int32, why string, rise float32, flat float32) {
	if !engine.DebugActions().Bool() {
		return
	}

	engine.PrintToServer("[teleclimb] %N %s, rise %.0f of %.0f to %.0f, out %.0f of %.0f, climb %d of %d",
		actor, why, rise, climbRiseMin, climbRiseMax,
		flat, climbRange, climbs[actor], climbLimit)
}

/*
ClimbToSpot crouch jumps onto the ground the spot sits on, and is false when there
is nothing to climb.

The stand point comes off the nav mesh, so for a spot on a rock the mesh does not
cover it is the floor underneath: he arrives, the spot is over his head, and every
placement from down there is refused. This puts him on top instead.

Once he is up, where he stands is where he stands. Recomputing the ring point from up
there asks the nav mesh again and the nav mesh answers with the floor he just left,
which is the walk back down. He climbed from within a build's reach, so the spot is
already in front of him.

The count resets when he makes it, so falling off and climbing again costs another
six attempts rather than none. The reach clock is what bounds the pair of them.
*/
//
//sp:name TeleporterClimbToSpot
func ClimbToSpot(actor int32, myBody engine.Body, spot [3]float32) bool {
	origin := engine.AbsOriginOf(actor)

	rise := spot[2] - origin[2]

	reach := engine.SubtractVectors(spot, origin)

	reach[2] = 0.0

	out := engine.VectorLength(reach)

	if rise < climbRiseMin {
		SayClimb(actor, "nothing to climb", rise, out)

		if climbs[actor] > 0 {
			climbs[actor] = 0
			standOf[actor] = origin
		}

		return false
	}

	// Higher than a crouch jump is not a ledge, it is a wall, and no number of jumps will do it
	if rise > climbRiseMax || climbs[actor] >= climbLimit {
		SayClimb(actor, engine.Choose(rise > climbRiseMax, "too high to climb", "out of climbs"), rise, out)

		return false
	}

	// Far enough out and the jump lands on the wall rather than on top of it
	if out > climbRange {
		SayClimb(actor, "too far out to climb", rise, out)

		return false
	}

	SayClimb(actor, "climbing", rise, out)

	engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.2, engine.AddressNull(), "Climbing to the teleporter spot")

	if climbAt[actor] > engine.GameTime() {
		return true
	}

	climbAt[actor] = engine.GameTime() + climbInterval
	climbs[actor]++

	// Forward is along where he is looking, which is the spot, so the three together are a person
	engine.ExtraButtonsOf(actor).PressButtons(engine.InForward()|engine.InJump()|engine.InDuck(), climbHold)

	return true
}

/*
FallBackToNest is the named exit spot having beaten him, so he takes the ring round
his own nest instead.

Once per action: the named flag is what selects it and this clears it, so there is
one fall back and then the ordinary give-up. False when there was no named spot to
fall back from.
*/
//
//sp:name TeleporterFallBackToNest
func FallBackToNest(actor int32) bool {
	if !namedSpot[actor] || mode[actor] != engine.ModeExit() {
		return false
	}

	namedSpot[actor] = false
	tryIndex[actor] = 0
	climbs[actor] = 0
	reachDeadline[actor] = engine.GameTime() + exitReachTime

	return StandPoint(actor)
}

/*
StandPoint is where this attempt puts the building, and where he stands to put it
there.

Three shapes, because the three cases are not the same question. A spot the map
named is one spot and the man walks round it. The way out of spawn is a route rather
than a spot, so the attempts walk along it, reading the points sampled off it when
the action started. The exit has no spot at all, only a nest, so the spot walks round
the nest and the man stands between the two.

False when this attempt has nowhere left to put anything, which is the caller's cue
to stop.
*/
//
//sp:name TeleporterStandPoint
func StandPoint(actor int32) bool {
	attempt := tryIndex[actor]

	if namedSpot[actor] {
		_, stand := engine.BuildStandPoint(spotOf[actor], engine.AbsOriginOf(actor), attempt,
			tryPoints, buildReach)

		standOf[actor] = stand

		return true
	}

	if mode[actor] == engine.ModeExit() {
		nest := nestOf[actor]

		// The safe ring first, the whole way round, then the tight one
		radius := float32(exitRadius)

		if attempt < tryPoints {
			radius = engine.BusterBlastRange() + 100.0
		}

		angle := attempt % tryPoints

		// Both on the same ray out of the nest, so he stands a build's reach short of the spot
		_, spot := engine.BuildStandPoint(nest, engine.AbsOriginOf(actor), angle,
			tryPoints, radius)

		spotOf[actor] = spot

		_, stand := engine.BuildStandPoint(nest, engine.AbsOriginOf(actor), angle,
			tryPoints, radius-buildReach)

		standOf[actor] = stand

		return true
	}

	if attempt >= routePoints[actor] {
		return false
	}

	spotOf[actor] = routeSpot[actor][attempt]
	standOf[actor] = routeStand[actor][attempt]

	return true
}

// Ended is every way this action can end, so the reason survives it.
//
//sp:name TeleporterDone
func Ended(action engine.Behaviour, actor int32, reason string) engine.Outcome {
	lastResult[actor] = engine.CopyText(reason)

	return action.EndWith(reason)
}

// OnEnd stops the walking.
func OnEnd(actor int32) {
	engine.PluginBotOf(actor).SetPathing(false)

	engine.UpdateLookAroundForEnemies(actor, true)
}

// HasGivenUp says he stopped asking for this round.
//
//sp:name EngineerTeleporter_HasGivenUp
func HasGivenUp(actor int32) bool {
	return gaveUp[actor]
}

// Mode is the half he is building.
//
//sp:name EngineerTeleporter_Mode
func Mode(actor int32) engine.ObjectMode {
	return mode[actor]
}

// Spot is where this attempt puts it.
//
//sp:name EngineerTeleporter_Spot
func Spot(actor int32) (spot [3]float32) {
	spot = spotOf[actor]

	return spot
}

// ForgetGivingUp is a new wave being a new chance, and whatever refused him last
// time may have been a body standing on it.
//
//sp:name EngineerTeleporter_ForgetGivingUp
func ForgetGivingUp() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		gaveUp[i] = false
	}
}

/*
ShouldBuild is the half of the teleporter this engineer should go build, or none.

Entrance before exit: an exit alone moves nobody, and the pair is only worth the
metal once both ends stand. The entrance spot comes from the map configuration when
it names one and from the way out of spawn when it does not, which is every official
map; the exit goes beside the nest.
*/
//
//sp:name ShouldBuildTeleporter
func ShouldBuild(actor int32) bool {
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return false
	}

	if gaveUp[actor] {
		return false
	}

	// The nest comes first and it is not finished
	// What is in his hands counts: a carried building is one he has, not one he needs
	if engine.HasObjectOfType(actor, engine.ObjectSentry(), engine.ModeNone()) == engine.InvalidEntReference() {
		return false
	}

	if engine.HasObjectOfType(actor, engine.ObjectDispenser(), engine.ModeNone()) == engine.InvalidEntReference() {
		return false
	}

	if engine.NestAreaOf(actor) == engine.NullArea() {
		return false
	}

	nestOf[actor] = engine.NestBuildPosition(engine.NestAreaOf(actor))

	if engine.ObjectOfTypeMode(actor, engine.ObjectTeleporter(), engine.ModeEntrance()) == engine.InvalidEntReference() {
		mode[actor] = engine.ModeEntrance()

		named, spot := engine.NearestConfiguredSpot(engine.TeleporterEntranceSpots(), engine.AbsOriginOf(actor))

		namedSpot[actor] = named
		spotOf[actor] = spot

		if namedSpot[actor] {
			return true
		}

		// The map named none, which is most of them, so he walks out of spawn until the floor takes it
		ok, spawn := engine.NearestSpawnPoint(actor)

		spawnOf[actor] = spawn

		return ok
	}

	if engine.ObjectOfTypeMode(actor, engine.ObjectTeleporter(), engine.ModeExit()) == engine.InvalidEntReference() {
		mode[actor] = engine.ModeExit()

		// The nest itself when the map names no exit: the point of the pair is to arrive at the nest
		named, spot := NearestFreeExitSpot(actor, nestOf[actor])

		namedSpot[actor] = named
		spotOf[actor] = spot

		return true
	}

	return false
}

/*
NearestFreeExitSpot is the nearest named exit spot another engineer has not already
put one on.

Coaltown names one exit and a team can field two engineers, and nothing stopped the
second from building his on top of the first: reported from play as two exits sitting
next to each other on the same platform. Two exits work, but the second one is a walk
and fifty metal spent arriving where somebody could already arrive.

With every named spot taken, this says no and the exit goes beside his own nest
instead, which is where an exit is for anyway. The dispenser has had this rule for a
while; the exit had not.
*/
//
//sp:name NearestFreeExitSpot
func NearestFreeExitSpot(actor int32, nest [3]float32) (found bool, spot [3]float32) {
	spots := engine.TeleporterExitSpots()

	if spots.Length() == 0 {
		return false, spot
	}

	free := engine.NewBlocks(3)
	defer free.Close()

	for i := int32(0); i < spots.Length(); i++ {
		candidate := spots.GetArray(i)

		if !IsExitSpotTaken(actor, candidate) {
			free.PushArray(candidate)
		}
	}

	found, spot = engine.NearestConfiguredSpot(free, nest)

	return found, spot
}

//sp:name TELEPORTER_EXIT_TAKEN_RANGE
const exitTakenRange = 200.0

// IsExitSpotTaken says somebody else's exit is already standing here.
//
//sp:name IsExitSpotTaken
func IsExitSpotTaken(actor int32, spot [3]float32) bool {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor || !engine.IsClientInGame(i) {
			continue
		}

		exitTele := engine.ObjectOfTypeMode(i, engine.ObjectTeleporter(), engine.ModeExit())

		if exitTele == engine.InvalidEntReference() {
			continue
		}

		if engine.VectorDistance(spot, engine.AbsOriginOf(exitTele)) < exitTakenRange {
			return true
		}
	}

	return false
}

// NearestSpot is false when the map names no spot of this kind, which is most of
// them.
//
//sp:name NearestConfiguredSpot
func NearestSpot(spots engine.List, from [3]float32) (found bool, spot [3]float32) {
	nearest := float32(-1.0)

	for i := int32(0); i < spots.Length(); i++ {
		candidate := spots.GetArray(i)

		distance := engine.VectorDistance(from, candidate)

		if nearest < 0.0 || distance < nearest {
			nearest = distance
			spot = candidate
		}
	}

	return nearest >= 0.0, spot
}
