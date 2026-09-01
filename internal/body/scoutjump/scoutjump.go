/*
Package scoutjump is the Scout's combat jump out of
source/redbots3/nextbot_behavior.sp: a hop while closing, on an irregular beat,
with a second half in the air often enough to be worth leading wrong.
*/
package scoutjump

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// ScoutJumpThreatRange is how close a threat has to be before jumping is worth
// the air time.
//
//sp:name SCOUT_JUMP_THREAT_RANGE
const ScoutJumpThreatRange = 900.0

// ScoutJumpMinSpeed is standing still: a jump in place lands where it started.
//
//sp:name SCOUT_JUMP_MIN_SPEED
const ScoutJumpMinSpeed = 100.0

// ScoutDoubleJumpChance is how often the hop gets its second half, per cent.
//
//sp:name SCOUT_DOUBLE_JUMP_CHANCE
const ScoutDoubleJumpChance = 70

// ScoutDoubleJumpDelay is how far into the hop the air jump fires.
//
//sp:name SCOUT_DOUBLE_JUMP_DELAY
const ScoutDoubleJumpDelay = 0.22

// ScoutJumpStrafeTime is how long the sideways key is held.
//
//sp:name SCOUT_JUMP_STRAFE_TIME
const ScoutJumpStrafeTime = 0.35

//sp:name m_flNextJumpTime
var nextJumpTime [Slots]float32

//sp:name m_flScoutDoubleJumpTime
var scoutDoubleJumpTime [Slots]float32

//sp:name m_iScoutDoubleJumpSide
var scoutDoubleJumpSide [Slots]int32

/*
UpdateScoutCombatJump is the whole dodge, run per think.

The second half of a double jump happens off the ground by definition, so it is
handled before every check that wants the bot standing on something. The beat is
irregular on purpose: a jump on a fixed rhythm is as easy to lead as no jump at
all, and the air jump goes the other way from the first strafe.
*/
//
//sp:name UpdateScoutCombatJump
func UpdateScoutCombatJump(client int32) {
	if engine.PlayerClass(client) != engine.ClassScout() {
		return
	}

	if scoutDoubleJumpTime[client] > 0.0 {
		if engine.GameTime() < scoutDoubleJumpTime[client] {
			return
		}

		scoutDoubleJumpTime[client] = 0.0

		// Landed early. The air jump is gone and pressing it again only
		// queues the next ground one.
		if engine.EntityFlags(client)&engine.FlagOnGround() != 0 {
			return
		}

		engine.ExtraButtonsOf(client).PressButtons(engine.InJump()|scoutDoubleJumpSide[client], ScoutJumpStrafeTime)

		return
	}

	if nextJumpTime[client] > engine.GameTime() {
		return
	}

	// Already in the air, or held down by something that a jump will not get
	// it out of.
	if engine.EntityFlags(client)&engine.FlagOnGround() == 0 {
		return
	}

	if engine.IsPlayerInCondition(client, engine.ConditionDazed()) || engine.IsPlayerInCondition(client, engine.ConditionSlowed()) {
		return
	}

	velocity := engine.EntPropVector(client, engine.PropData(), "m_vecVelocity")

	if velocity[0]*velocity[0]+velocity[1]*velocity[1] < ScoutJumpMinSpeed*ScoutJumpMinSpeed {
		return
	}

	myBot := engine.NextBotOf(client)
	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat == engine.NoKnownEntity() || !threat.VisibleRecently() {
		return
	}

	threatOrigin := threat.LastKnownPosition()

	if myBot.IsRangeGreaterThanEx(threatOrigin, ScoutJumpThreatRange) {
		return
	}

	// Irregular on purpose: a jump on a fixed beat is as easy to lead as no
	// jump at all.
	nextJumpTime[client] = engine.GameTime() + engine.RandomFloat(0.5, 1.2)

	side := engine.InMoveLeft()

	if engine.RandomInt(0, 1) == 0 {
		side = engine.InMoveRight()
	}

	engine.ExtraButtonsOf(client).PressButtons(engine.InJump()|side, ScoutJumpStrafeTime)

	if engine.RandomInt(1, 100) > ScoutDoubleJumpChance {
		return
	}

	scoutDoubleJumpTime[client] = engine.GameTime() + ScoutDoubleJumpDelay

	if side == engine.InMoveLeft() {
		scoutDoubleJumpSide[client] = engine.InMoveRight()
	} else {
		scoutDoubleJumpSide[client] = engine.InMoveLeft()
	}
}

// ResetScoutJump forgets this scout's jump beat.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetScoutJump(client int32) {
	nextJumpTime[client] = 0.0
}
