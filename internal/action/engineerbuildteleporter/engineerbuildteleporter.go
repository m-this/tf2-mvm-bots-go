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

//sp:action DefenderBuildTeleporter CTFBotMvMEngineerBuildTeleporter
*/
package engineerbuildteleporter

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/climb"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/nestsetup"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/nestspot"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

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
ExitRadius is the tight ring: the exit stands off the nest centre rather than on it

The nest centre is where the sentry is, so eight stand points looking at the centre
are eight looks at the sentry and eight refusals. The spot walks round the nest
instead, and he stands between the two: far enough out not to be inside his own
sentry, a build's reach short of where the exit goes.
*/
//
//sp:name TELEPORTER_EXIT_RADIUS
const ExitRadius = 150.0

// ExitRadiusSafe is the ring tried first, sized off BUSTER_BLAST_RANGE with a
// hundred to spare rather than off the build reach. internal/tables holds the
// relation.
//
//sp:name TELEPORTER_EXIT_RADIUS_SAFE
const ExitRadiusSafe = 500.0

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
	giveUp [slots.Count]float32
	//sp:name m_ctTeleporterReachDeadline
	reachDeadline [slots.Count]float32
	//sp:name m_ctTeleporterTryDeadline
	tryDeadline [slots.Count]float32
	//sp:name m_iTeleporterTry
	tryIndex [slots.Count]int32
	//sp:name m_nTeleporterMode
	mode [slots.Count]engine.ObjectMode
	//sp:name m_vTeleporterSpot
	spotOf [slots.Count][3]float32
	//sp:name m_vTeleporterStand
	standOf [slots.Count][3]float32
	//sp:name m_vTeleporterSpawn
	spawnOf [slots.Count][3]float32
	//sp:name m_vTeleporterNest
	nestOf [slots.Count][3]float32
	/* The way out of spawn, read once while he is still standing at the far end of it

	Read per attempt instead, it was read from wherever he had walked to, and the second attempt asked
	a two hundred and ninety unit route for a point three hundred and fifty units from spawn. He tried
	one place on Coaltown and gave up. */
	//
	//sp:name m_vTeleporterRouteSpot
	routeSpot [slots.Count][tryPoints][3]float32
	//sp:name m_vTeleporterRouteStand
	routeStand [slots.Count][tryPoints][3]float32
	//sp:name m_iTeleporterRoutePoints
	routePoints [slots.Count]int32
	// The map named the spot, so the attempts walk around it instead of out of the spawn door
	//
	//sp:name m_bTeleporterNamedSpot
	namedSpot [slots.Count]bool
	/* He tried everything and none of it worked, so he stops asking until the next wave is over

	Without this the idle action suspends into this one again the moment it ends, which is an engineer
	walking the same refused route for the rest of the round, and readiness waiting on him while he
	does it. */
	//
	//sp:name m_bTeleporterGaveUp
	gaveUp [slots.Count]bool
	/* This attempt is the entrance on the way out of spawn, before the nest

	It has its own give-up, so an early attempt that finds nowhere leaves the ordinary one, after the
	nest, untouched: the fault the third A/B on mvm-dh8 measured was one refusal latching gaveUp for
	the whole break. */
	//
	//sp:name m_bTeleporterEntranceFirst
	entranceFirst [slots.Count]bool
	//sp:name m_bTeleporterEntranceFirstTried
	entranceFirstTried [slots.Count]bool
	// Why the last attempt ended, for sm_dump_nest, since every give-up looks the same from outside
	//
	//sp:name m_sTeleporterLastResult
	lastResult [slots.Count]engine.Text
)

// OnStart reads the route out of spawn while he is still standing at his nest.
func OnStart(actor int32) engine.Outcome {
	giveUp[actor] = engine.GameTime() + buildMaxTime
	reachDeadline[actor] = engine.GameTime() + exitReachTime
	tryDeadline[actor] = engine.GameTime() + tryTime
	climb.Begin(actor)
	tryIndex[actor] = 0
	routePoints[actor] = 0

	// From the nest towards spawn when he stands at the nest, from spawn towards the nest when he
	// is still in it: the same route, read from whichever end he is at
	if mode[actor] == engine.ModeEntrance() && !namedSpot[actor] {
		if entranceFirst[actor] {
			routePoints[actor], routeSpot[actor], routeStand[actor] = engine.SpawnRouteOut(actor, nestOf[actor],
				spawnOffset, spawnStep, buildReach)
		} else {
			routePoints[actor], routeSpot[actor], routeStand[actor] = engine.SpawnRoutePoints(actor, spawnOf[actor],
				spawnOffset, spawnStep, buildReach)
		}
	}

	if !StandPoint(actor) {
		GiveUp(actor)

		return Ended(engine.ThisAction(), actor, "No route out of spawn to walk")
	}

	/* Priced by the walk, because the exit is built after the entrance and the entrance is at spawn

	Rottenburg's exit spot is 3300 units from its spawn door, which is more than the flat twelve
	seconds, so the clock ran out on the way and the exit went to the nest ring every time. The
	dispenser learned the same lesson on Coaltown. */
	reachDeadline[actor] = engine.GameTime() + engine.BuildReachTime(engine.AbsOriginOf(actor), standOf[actor])

	/* The half he is about to build is claimed, and the walk to it is a jump

	This is the walk the whole complaint was about: the entrance is at the far end of the map from
	the nest, so building the pair costs the length of the map twice. The jump refuses itself
	during a wave, so what a wave sees is the walk. */
	if engine.Feature(engine.FeatureEngineerSetupPhase()) {
		nestsetup.ClaimSetupSpot(actor, TeleporterClaim(actor), spotOf[actor])
		nestsetup.SetupJump(actor, standOf[actor])
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

	if !entranceFirst[actor] && engine.ObjectOfType(actor, engine.ObjectSentry()) == engine.InvalidEntReference() {
		return Ended(engine.ThisAction(), actor, "No sentry to leave behind")
	}

	built := engine.ObjectOfTypeMode(actor, engine.ObjectTeleporter(), mode[actor])

	if built != engine.InvalidEntReference() {
		nestsetup.SayIfBuiltElsewhere(actor, built, spotOf[actor], "teleporter")
		engine.PluginBotOf(actor).SetPathing(false)

		return Ended(engine.ThisAction(), actor, "Built one")
	}

	if giveUp[actor] < engine.GameTime() {
		GiveUp(actor)

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

	// The map put the spot on top of something, so he gets on top of it rather than building below it
	if !outOfTime && namedSpot[actor] {
		climbed := climb.ToSpot(actor, myBody, spot, "teleporter spot")

		if climbed == climb.Busy {
			engine.PluginBotOf(actor).SetPathing(false)

			return engine.Continue()
		}

		/* The reach clock starts again on landing, because six jumps and a lift are most of it: on
		Rottenburg the exit was lifted onto its spot and then fell back to the nest ring on a clock
		that had run out while he was jumping. */
		if climbed == climb.Landed {
			reachDeadline[actor] = engine.GameTime() + exitReachTime
		}

		// Up on the rock nothing is pathed to: where he stands is the stand point, stepped off the spot
		if climb.OnTop(actor) && climb.Beside(actor, spot) {
			standOf[actor] = engine.AbsOriginOf(actor)

			if climb.StepBack(actor, myBody, spot) {
				engine.PluginBotOf(actor).SetPathing(false)

				return engine.Continue()
			}
		}
	}

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
		engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.1, engine.NoAddress(), "Placing teleporter")
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
					GiveUp(actor)

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
	climb.Begin(actor)
	reachDeadline[actor] = engine.GameTime() + exitReachTime

	return StandPoint(actor)
}

/*
	How far a player could fall stepping off the exit, and the side that avoids it

A nest that relocates has its exit ring placed round it, eight sides at
TELEPORTER_EXIT_RADIUS_SAFE, and nothing looked down. A player taking the
teleporter is put somewhere the engineer never stood: mvm-1yo, and mvm-0am is
the report of it on Rottenburg.

Measured offline against the nav mesh, the declared nests are almost all clear:
of seventeen on the seven maps with a mesh, only two Mannhattan nests have a
side beside a hurting fall at all, and both keep a safe side. It is the
relocated nest, on ground nobody declared, that this is for.

A preference and never a refusal. The sides are walked from the one the attempt
asked for, the first without a hurting fall wins, and a ring where every side
falls keeps the side it would have had.
*/

// FallHurtHeight is the drop at which Team Fortress 2 starts charging fall
// damage. It is internal/navmesh's FallDamageHeight, and a test holds the two
// together.
//
//sp:name TELEPORTER_EXIT_FALL_HURT
const FallHurtHeight = 264.0

// ExitDropRadius is how far round the exit a player might step before falling,
// and ExitDropDepth is how far down the trace looks. internal/navmesh uses the
// same radius.
//
//sp:name TELEPORTER_EXIT_DROP_RADIUS
const ExitDropRadius = 150.0

// ExitDropDepth is the length of the ray, which has to outrun the worst fall a
// map has rather than the worst that hurts.
//
//sp:name TELEPORTER_EXIT_DROP_DEPTH
const ExitDropDepth = 2000.0

// ExitDropEye is where the ray starts above the point, so it does not begin
// inside the floor.
//
//sp:name TELEPORTER_EXIT_DROP_EYE
const ExitDropEye = 36.0

// SideWithoutAFall is the first ring side from wanted whose exit has no hurting
// fall beside it, or wanted when every side has one.
//
//sp:name SideWithoutAFall
func SideWithoutAFall(actor int32, nest [3]float32, wanted int32, radius float32) int32 {
	for step := int32(0); step < tryPoints; step++ {
		side := (wanted + step) % tryPoints

		_, spot := engine.BuildStandPoint(nest, engine.AbsOriginOf(actor), side, tryPoints, radius)

		if WorstDropAround(spot) < FallHurtHeight {
			return side
		}
	}

	return wanted
}

/*
WorstDropAround is the deepest fall a player could take stepping off this point.

Five rays: the point itself and four steps out at the radius a player could
cover before the floor stops holding them. That is what internal/navmesh's
CheckDrop reads off the mesh, done with the engine's own traces because a
relocated nest is not in any config for the mesh model to have looked at.
*/
//
//sp:name WorstDropAround
//sp:const at
func WorstDropAround(at [3]float32) float32 {
	worst := float32(0)

	for probe := int32(0); probe < 5; probe++ {
		var from [3]float32

		from[0] = at[0] + DropProbeX(probe)*ExitDropRadius
		from[1] = at[1] + DropProbeY(probe)*ExitDropRadius
		from[2] = at[2] + ExitDropEye

		var to [3]float32

		to[0] = from[0]
		to[1] = from[1]
		to[2] = from[2] - ExitDropDepth

		engine.TraceRay(from, to, engine.MaskPlayerSolid(), engine.RayTypeEndPoint())

		ground := engine.TraceEndPosition()
		drop := from[2] - ground[2]

		if drop > worst {
			worst = drop
		}
	}

	return worst
}

// DropProbeX and DropProbeY are the five probes: the point, then north, east,
// south and west of it. Written as a switch because the subset has no table of
// vectors to index.
//
//sp:name DropProbeX
func DropProbeX(probe int32) float32 {
	switch probe {
	case 1:
		return 1.0
	case 3:
		return -1.0
	}

	return 0.0
}

// DropProbeY is the other half of the same five.
//
//sp:name DropProbeY
func DropProbeY(probe int32) float32 {
	switch probe {
	case 2:
		return 1.0
	case 4:
		return -1.0
	}

	return 0.0
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
		// A side level with the spot before a side a storey under it, see LevelStandPoint
		_, side, stand := nestspot.LevelStandPoint(spotOf[actor], engine.AbsOriginOf(actor), attempt,
			tryPoints, buildReach)

		tryIndex[actor] = side
		standOf[actor] = stand

		return true
	}

	if mode[actor] == engine.ModeExit() {
		nest := nestOf[actor]

		// The safe ring first, the whole way round, then the tight one
		radius := float32(ExitRadius)

		if attempt < tryPoints {
			radius = ExitRadiusSafe
		}

		angle := SideWithoutAFall(actor, nest, attempt%tryPoints, radius)

		// Both on the same ray out of the nest, so he stands a build's reach short of the spot
		_, spot := engine.BuildStandPoint(nest, engine.AbsOriginOf(actor), angle,
			tryPoints, radius)

		spotOf[actor] = spot

		_, stand := engine.BuildStandPoint(nest, engine.AbsOriginOf(actor), angle,
			tryPoints, radius-buildReach)

		standOf[actor] = stand

		return true
	}

	/* Past whatever another engineer has claimed, rather than onto it

	The route out of spawn is the same route for everybody who spawns there, so two engineers
	reading it pick the same first point and stand in each other. The points are a hundred and
	fifty apart and there are eight of them, so stepping past a claim costs a step. */
	for ; attempt < routePoints[actor]; attempt++ {
		if engine.Feature(engine.FeatureEngineerSetupPhase()) &&
			nestsetup.IsSetupSpotClaimed(actor, routeSpot[actor][attempt]) {
			continue
		}

		tryIndex[actor] = attempt
		spotOf[actor] = routeSpot[actor][attempt]
		standOf[actor] = routeStand[actor][attempt]

		return true
	}

	return false
}

// TeleporterClaim is which of the four setup spots this attempt is for.
//
//sp:name TeleporterClaim
func TeleporterClaim(actor int32) int32 {
	if mode[actor] == engine.ModeExit() {
		return nestsetup.SetupExit
	}

	return nestsetup.SetupEntrance
}

// GiveUp stops the asking: for the break when the nest stood, for the early
// attempt alone when it did not, so the ordinary one still gets its turn.
//
//sp:name TeleporterGiveUp
func GiveUp(actor int32) {
	if entranceFirst[actor] {
		entranceFirstTried[actor] = true

		return
	}

	gaveUp[actor] = true
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

/*
LastResult is why the last attempt ended, copied into the buffer sm_dump_nest
handed in.

//sp:name EngineerTeleporter_LastResult
//sp:length buffer maxlength
*/
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size, which Go has no way to say
func LastResult(actor int32, buffer engine.Text, maxlength int32) {
	buffer = engine.CopyTextInto(engine.ChooseText(lastResult[actor][0] == 0, "nothing yet", lastResult[actor]))
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

// ResetBuildTeleporter forgets the early entrance for a seat, since the next bot
// in it is a different bot. Nothing else here is touched: the fields that were
// already carried between bots stay on the unreviewed list until mvm-z83.91
// decides them.
//
//sp:name Go_ResetBuildTeleporter
func ResetBuildTeleporter(client int32) {
	entranceFirst[client] = false
	entranceFirstTried[client] = false
}

// ForgetGivingUp is a new wave being a new chance, and whatever refused him last
// time may have been a body standing on it.
//
//sp:name EngineerTeleporter_ForgetGivingUp
func ForgetGivingUp() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		gaveUp[i] = false
		entranceFirst[i] = false
		entranceFirstTried[i] = false
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

	entranceFirst[actor] = false

	// The entrance first, while he still stands in spawn, when the switch says so: the nest is
	// picked by then and he is teleported onto it the moment the sentry action starts, so the
	// only walk this costs is the few hundred units out of the door
	if engine.Feature(engine.FeatureEngineerEntranceFirst()) && !entranceFirstTried[actor] &&
		engine.HasObjectOfType(actor, engine.ObjectSentry(), engine.ModeNone()) == engine.InvalidEntReference() &&
		engine.ObjectOfTypeMode(actor, engine.ObjectTeleporter(), engine.ModeEntrance()) == engine.InvalidEntReference() {
		return ShouldBuildEntranceFirst(actor)
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

// ShouldBuildEntranceFirst is the early entrance: the nest must be picked, since
// the route out of spawn is read towards it, and the map's own spot wins when it
// names one.
//
//sp:name ShouldBuildEntranceFirst
func ShouldBuildEntranceFirst(actor int32) bool {
	if engine.NestAreaOf(actor) == engine.NullArea() {
		return false
	}

	nestOf[actor] = engine.NestBuildPosition(engine.NestAreaOf(actor))
	mode[actor] = engine.ModeEntrance()

	named, spot := engine.NearestConfiguredSpot(engine.TeleporterEntranceSpots(), engine.AbsOriginOf(actor))

	namedSpot[actor] = named
	spotOf[actor] = spot

	if namedSpot[actor] {
		entranceFirst[actor] = true

		return true
	}

	ok, spawn := engine.NearestSpawnPoint(actor)

	spawnOf[actor] = spawn
	entranceFirst[actor] = ok

	return ok
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

		if engine.Feature(engine.FeatureEngineerSetupPhase()) && nestsetup.IsSetupSpotClaimed(actor, candidate) {
			continue
		}

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
