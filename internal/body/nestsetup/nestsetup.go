/*
Package nestsetup is the engineer's setup between waves.

A break is a fixed clock and the engineer spent it walking: out to the nest, back
to spawn for the entrance, out again for the exit, and a wrench on each until it
read level three. Measured on Decoy at fifteen thousand units a break, with the
sentry still at level one or two in one sample of every seven.

So a break is a plan rather than a walk. Every spot is claimed before anybody
goes anywhere, so two engineers never build on top of one another; he jumps to
each spot instead of walking to it; and what he puts down comes up finished.

Nothing here runs during a wave, and the round state is checked in every entry
point rather than by the callers. A jump mid-wave is a cheat and a way into
geometry both, and a building that arrives finished mid-wave is a different game.
*/
package nestsetup

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/spawnexit"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
The four things an engineer puts down, in the order he puts them down.

They are an index into his claims rather than a TFObjectType because the two
halves of the teleporter are one type and two spots, and it is the spots that
have to be kept apart.
*/
const (
	//sp:name SETUP_SENTRY
	SetupSentry = 0
	//sp:name SETUP_DISPENSER
	SetupDispenser = 1
	//sp:name SETUP_ENTRANCE
	SetupEntrance = 2
	//sp:name SETUP_EXIT
	SetupExit = 3
	//sp:name SETUP_SPOTS
	setupSpots = 4
)

/*
How close another engineer's claim is too close

Two hundred, which is what the teleporter exit already refused another exit at. A
building is about seventy units across and an engineer standing at one is another
forty, so this is the pair of them with room to walk between rather than a tight
fit.
*/
//
//sp:name SETUP_SPOT_SPACING
const setupSpotSpacing = 200.0

/*
What one level of a building costs in metal

Two hundred, which is what the game charges and what a wrench pays off twenty
five at a time. Topping the meter up to it does not upgrade anything by itself:
the game applies the level when the engineer hits the building, so this is the
difference between eight hits a level and one.
*/
//
//sp:name SETUP_UPGRADE_COST
const setupUpgradeCost = 200

/*
How many jumps a break may spend

Four buildings and four retries. A cap rather than a clock because the failure
this bounds is a spot that refuses every placement: without it the action ends,
starts again on the next tick, and jumps him for the rest of the break.
*/
//
//sp:name SETUP_JUMPS_MAX
const setupJumpsMax = 8

var (
	//sp:name m_vSetupClaim
	claim [slots.Count][setupSpots][3]float32
	//sp:name m_bSetupClaimed
	claimed [slots.Count][setupSpots]bool
	//sp:name m_iSetupJumps
	jumps [slots.Count]int32
)

/*
ResetNestSetup forgets a seat's plan, which the next bot in it did not make.

Also the whole of what a new wave does to a plan: the spots were claimed against
a break that is over, and the buildings that stand are claim enough on their own.
*/
//
//sp:name Go_ResetNestSetup
func ResetNestSetup(client int32) {
	for what := int32(0); what < setupSpots; what++ {
		claimed[client][what] = false
	}

	jumps[client] = 0
}

// ForgetSetupPlans is every seat's plan at once, for the start of a wave.
//
//sp:name ForgetSetupPlans
func ForgetSetupPlans() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		ResetNestSetup(i)
	}
}

/*
ClaimSetupSpot says this engineer means to build there, before he has.

The claim is what another engineer reads. A building that already stands says the
same thing and says it better, so claims are not kept once the thing is up: they
are cleared with the plan at the start of the wave.
*/
//
//sp:name ClaimSetupSpot
//sp:const spot
func ClaimSetupSpot(client int32, what int32, spot [3]float32) {
	if what < 0 || what >= setupSpots {
		return
	}

	claim[client][what] = spot
	claimed[client][what] = true
}

/*
IsSetupSpotClaimed says another engineer got there first.

Only another engineer's claims, and only the ones he has not built yet: what is
standing is already refused by the checks each building has of its own, and
counting it twice would refuse an engineer the ground he is himself standing on.
*/
//
//sp:name IsSetupSpotClaimed
//sp:const spot
func IsSetupSpotClaimed(client int32, spot [3]float32) bool {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client || !engine.IsClientInGame(i) {
			continue
		}

		for what := int32(0); what < setupSpots; what++ {
			if !claimed[i][what] {
				continue
			}

			if engine.VectorDistance(spot, claim[i][what]) < setupSpotSpacing {
				return true
			}
		}
	}

	return false
}

/*
SetupJump puts the engineer at a spot he would otherwise walk to.

Every bound is here rather than at the call sites: between rounds, alive, a
defender engineer, a destination with room to stand, and a fixed number of them
per break. The destination is traced because the nest and the route out of spawn
are both points on a nav mesh, and a nav mesh says ground is connected without
promising a body fits there. That is how mvm-qhi hung a server.

False means he walks, which is what he did before this existed.
*/
//
//sp:name SetupJump
//sp:const spot
func SetupJump(client int32, spot [3]float32) bool {
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return false
	}

	if !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
		return false
	}

	if engine.PlayerClass(client) != engine.ClassEngineer() || !engine.DefenderBotFlag(client) {
		return false
	}

	if jumps[client] >= setupJumpsMax {
		return false
	}

	destination := spot
	destination[2] += engine.StepHeight()

	if !spawnexit.IsRoomToStand(destination) {
		return false
	}

	jumps[client]++
	engine.EntityOf(client).SetAbsOrigin(destination)

	return true
}

/*
TopUpUpgrades pays for the levels the engineer is about to swing for.

The wrench is what upgrades a building and nothing else does: writing the level
is a number with a level one model, a level one health pool and a level one
firing rate behind it, and writing m_iHighestUpgradeLevel does nothing at all,
which two runs on Decoy said plainly. So the meter is filled instead and the
swing he was going to make anyway finishes the level.

Between rounds only, and the round state is checked here so the caller can be a
single line in the idle behaviour. During a wave a building is upgraded the way
it always was.
*/
//
//sp:name TopUpUpgrades
func TopUpUpgrades(client int32) {
	if !engine.Feature(engine.FeatureEngineerSetupPhase()) {
		return
	}

	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return
	}

	if !engine.DefenderBotFlag(client) {
		return
	}

	TopUpBuilding(engine.ObjectOfType(client, engine.ObjectSentry()))
	TopUpBuilding(engine.ObjectOfType(client, engine.ObjectDispenser()))
	TopUpBuilding(engine.ObjectOfTypeMode(client, engine.ObjectTeleporter(), engine.ModeEntrance()))
	TopUpBuilding(engine.ObjectOfTypeMode(client, engine.ObjectTeleporter(), engine.ModeExit()))
}

/*
TopUpBuilding pays for the level the engineer is about to swing for.

The meter is filled first and always: that is what shipped, and one wrench swing
is then worth a whole level. CBaseObject::StartUpgrading is asked for the level
outright afterwards, where the gamedata reaches it, because writing
m_iHighestUpgradeLevel does nothing at all.

The call is not trusted on its own. Measured on Decoy 2026-09-06, one wave, the
sentry read level 1 in all 22 samples with the call in place of the meter, so on
this build it applied nothing and the meter is what works. It is called after
the metal rather than instead of it, so a build where it does work loses no
wrench swing and one where it does not loses nothing at all.

A mini has no upgrade path and one that is still going up has not got a meter to
fill yet: the game clears it when the construction finishes.
*/
//
//sp:name TopUpBuilding
func TopUpBuilding(building int32) {
	if building == engine.InvalidEntReference() || !engine.IsValidEntity(building) {
		return
	}

	if engine.IsMiniBuilding(building) || engine.IsBuildingUp(building) {
		return
	}

	if engine.UpgradeLevel(building) >= engine.MaxUpgradeLevel() {
		return
	}

	engine.SetEntPropSend(building, engine.PropSend(), "m_iUpgradeMetal", setupUpgradeCost)

	if engine.CallStartUpgrading() != engine.NoCall() {
		engine.StartUpgrading(building)
	}
}

/*
	When a building lands somewhere other than the spot it was aimed at

A spot on top of a rock has every stand side accepted and every one of them a
storey below it, so the engineer builds at the foot of the rock and the config
still reads as if it worked. Three Bigrock spots and one on Coaltown are like
that, measured against the nav mesh, and the only symptom anybody ever saw was
a player noticing a dispenser in the wrong place. That is mvm-wxp.

This does not move the building. It says the thing out loud, once per build, so
a map config that quietly does nothing stops reading like one that works.
*/

// BuildStrayDistance is how far a building may land from the spot it was aimed
// at before it is worth a line. A build's reach is 90, so anything past twice
// that is a different piece of ground rather than the same one.
//
//sp:name BUILD_STRAY_DISTANCE
const BuildStrayDistance = 180.0

// SayIfBuiltElsewhere names a building that went down away from its spot.
//
//sp:name SayIfBuiltElsewhere
//sp:const spot
func SayIfBuiltElsewhere(client int32, building int32, spot [3]float32, what string) {
	if building == engine.InvalidEntReference() || !engine.IsValidEntity(building) {
		return
	}

	away := engine.VectorDistance(engine.AbsOriginOf(building), spot)

	if away < BuildStrayDistance {
		return
	}

	engine.LogMessage("Placement: %N put his %s %.0f units from the spot he aimed at, which is mvm-wxp",
		client, what, away)
}
