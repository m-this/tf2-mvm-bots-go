/*
Package stickytrap is source/redbots3/behavior/stickytrap.sp.

Laying a stickybomb trap, which is the half of the Demoman that firing the
launcher is not.

The state machine is Cheeseh's, from RCBot2's CBotTF2::deployStickies, and it is
worth copying because it is the small honest version of a thing that invites a
large dishonest one. A bot does not need to know where the robots will walk. It
needs a piece of ground, a handful of bombs stacked on it, a gap between shots so
the launcher keeps up, and somebody to decide when the ground is worth it.

	the ground    for a defender, where the bomb is. Robots escort it, so it is
	              the one place on the map they are all walking to, and it is
	              where the carrier will stand
	the stack     a small scatter around one point, so a giant standing on it
	              takes all eight bombs rather than the two it walked near
	the gap       a second or so between shots, which is what the launcher wants
	              and what stops the bot emptying a clip into a wall while it
	              turns
	the deadline  because a Demoman standing in the open aiming at the floor is
	              a Demoman not fighting, and the wave does not wait for him

Detonation is not here. ShouldDetonateStickies reads where the bombs actually
landed, which is better than trusting where they were aimed.

//sp:action DefenderStickyTrap CTFBotStickyTrap
*/
package stickytrap

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// The launcher holds eight, and eight is the trap.
//
//sp:name STICKY_TRAP_BOMBS
const trapBombs = 8

/*
How wide to scatter them, and why it is this narrow.

It was 120, roughly a blast across, on the reasoning that a spread covers ground
without gaps. That is the right trap for a crowd and the wrong one for what
actually walks into it here. Every guide written about this class says the same
thing: stack the bombs on one spot for a giant, and carpet only for a group or a
line of Medics.
*/
//
//sp:name STICKY_TRAP_SPREAD
const trapSpread = 40.0

//sp:name STICKY_TRAP_CARPET
const trapCarpet = 120.0

// What the launcher wants between shots, and what the bot needs to turn between
// them.
const (
	//sp:name STICKY_TRAP_SHOT_GAP_MIN
	shotGapMin = 0.6
	//sp:name STICKY_TRAP_SHOT_GAP_MAX
	shotGapMax = 0.9
)

// A trap is worth this many seconds of a wave and no more.
//
//sp:name STICKY_TRAP_MAX_TIME
const trapMaxTime = 12.0

// Nearer than this and the bot is standing in its own trap. Further and it is
// aiming at a rumour.
const (
	//sp:name STICKY_TRAP_MIN_RANGE
	trapMinRange = 350.0
	//sp:name STICKY_TRAP_MAX_RANGE
	trapMaxRange = 1500.0
)

// Bombs already down that make another trap a waste of the clip.
//
//sp:name STICKY_TRAP_ENOUGH
const trapEnough = 4

// How long before the same bot bothers again, so a Demoman is not permanently
// gardening.
//
//sp:name STICKY_TRAP_COOLDOWN
const trapCooldown = 20.0

var (
	//sp:name m_ctStickyTrapEnd
	trapEnd [slots.Count]float32
	//sp:name m_ctStickyTrapNextShot
	trapNextShot [slots.Count]float32
	//sp:name m_ctStickyTrapAgain
	trapAgain [slots.Count]float32
	//sp:name m_iStickyTrapBombsLeft
	trapBombsLeft [slots.Count]int32
	//sp:name m_vStickyTrapSpot
	trapSpot [slots.Count][3]float32
	//sp:name m_vStickyTrapPoint
	trapPoint [slots.Count][3]float32
)

// OnStart picks the ground and counts the bombs.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	trapEnd[actor] = engine.GameTime() + trapMaxTime
	trapNextShot[actor] = 0.0
	trapPoint[actor] = engine.NullVector()

	launcher := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())
	bombs := int32(0)

	if launcher != -1 {
		bombs = engine.EntProp(launcher, engine.PropData(), "m_iClip1")
	}

	trapBombsLeft[actor] = trapBombs

	if bombs < trapBombs {
		trapBombsLeft[actor] = bombs
	}

	found, spot := Spot()

	trapSpot[actor] = spot

	if !found {
		trapBombsLeft[actor] = 0
	}

	engine.SpeakConceptIfAllowed(actor, engine.ConceptSentryHere())

	return engine.Continue()
}

// Update walks in, aims at the floor and fires one bomb at a time.
func Update(actor int32) engine.Outcome {
	trapAgain[actor] = engine.GameTime() + trapCooldown

	if trapBombsLeft[actor] <= 0 {
		return engine.Done("Trap is laid")
	}

	if trapEnd[actor] < engine.GameTime() {
		return engine.Done("Took too long to lay a trap")
	}

	launcher := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if launcher == -1 || engine.WeaponID(launcher) != engine.WeaponPipebombLauncher() || !engine.HasAmmo(launcher) {
		return engine.Done("Nothing to lay it with")
	}

	myBot := engine.NextBotOf(actor)

	/* Something is shooting at the bot, so the trap stops being the job
	The bombs already down are not wasted: the detonation tick blows them the moment the fight
	walks into them, whether this action laid all eight or two */
	threat := myBot.Vision().PrimaryKnownThreat(true)

	if threat != engine.NoKnownEntity() {
		return engine.Done("Something to fight")
	}

	myOrigin := engine.Origin(actor)
	trapRange := engine.VectorDistance(myOrigin, trapSpot[actor])

	// Too far to aim at the ground honestly. Walk in, and give up if the walk takes the deadline
	if trapRange > trapMaxRange {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 0.4))
			engine.RepathToPos(actor, myBot, trapSpot[actor])
		}

		engine.PathOf(actor).Update(myBot)

		return engine.Continue()
	}

	// Standing in it. Anywhere further from the trap will do, and the path back is the way it came
	if trapRange < trapMinRange {
		return engine.Done("Standing in my own trap")
	}

	engine.SetPlayerActiveWeapon(actor, launcher)

	// A fresh point for each bomb, near enough to the last that a giant takes the whole stack
	if engine.IsZeroVector(trapPoint[actor]) {
		spread := float32(trapCarpet)

		if engine.Feature(engine.FeatureStickyStack()) {
			spread = trapSpread
		}

		trapPoint[actor][0] = trapSpot[actor][0] + engine.RandomFloat(-spread, spread)
		trapPoint[actor][1] = trapSpot[actor][1] + engine.RandomFloat(-spread, spread)
		trapPoint[actor][2] = trapSpot[actor][2]
	}

	myBody := myBot.Body()

	engine.AimHeadTowards(myBody, trapPoint[actor], engine.AimCritical(), 1.0, engine.NoAddress(), "Laying a sticky trap")

	if trapNextShot[actor] > engine.GameTime() {
		return engine.Continue()
	}

	if !myBody.IsHeadAimingOnTarget() {
		return engine.Continue()
	}

	engine.PressFireButton(actor)

	trapNextShot[actor] = engine.GameTime() + engine.RandomFloat(shotGapMin, shotGapMax)
	trapPoint[actor] = engine.NullVector()
	trapBombsLeft[actor]--

	return engine.Continue()
}

/*
Spot is the ground worth trapping, which for a defender is wherever the bomb is.

Robots escort it, so it is the one piece of ground every robot on the map is
walking towards, and the carrier stands on it while the rest of them fight around
it. With no bomb in play the hatch is the same argument with the robots not there
yet.
*/
//
//sp:name StickyTrapSpot
func Spot() (found bool, spot [3]float32) {
	haveBomb, bombinfo := engine.GetBombInfo()

	if haveBomb {
		spot = bombinfo.Position

		return true, spot
	}

	spot = engine.BombHatchPosition()

	return !engine.IsZeroVector(spot), spot
}

// Reset forgets the trap, which is what a death or a wave end does.
//
//sp:name ResetStickyTrap
func Reset(client int32) {
	trapAgain[client] = 0.0
	trapBombsLeft[client] = 0
	trapSpot[client] = engine.NullVector()
}

// IsPossible is the seven questions asked before a Demoman stops fighting to
// lay one.
//
//sp:name CTFBotStickyTrap_IsPossible
func IsPossible(client int32) bool {
	if engine.PlayerClass(client) != engine.ClassDemoMan() {
		return false
	}

	if trapAgain[client] > engine.GameTime() {
		return false
	}

	launcher := engine.PlayerWeaponSlot(client, engine.WeaponSlotSecondary())

	if launcher == -1 || engine.WeaponID(launcher) != engine.WeaponPipebombLauncher() {
		return false
	}

	if !engine.HasAmmo(launcher) || engine.EntProp(launcher, engine.PropData(), "m_iClip1") <= 0 {
		return false
	}

	// There is already a trap down. Another one is the same ground covered twice
	if engine.EntProp(launcher, engine.PropSend(), "m_iPipebombCount") >= trapEnough {
		return false
	}

	// A fight is not the time. Laying a trap is what a Demoman does before one
	if engine.NextBotOf(client).Vision().PrimaryKnownThreat(true) != engine.NoKnownEntity() {
		return false
	}

	found, spot := Spot()

	if !found {
		return false
	}

	myOrigin := engine.Origin(client)
	trapRange := engine.VectorDistance(myOrigin, spot)

	return trapRange > trapMinRange && trapRange < trapMaxRange
}
