/*
Package movement is the part of source/redbots3/util.sp that pushes a bot at a
goal and reads what its weapon is charged to.
*/
package movement

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// GetCurrentCharge is how far a chargeable weapon has wound up, from zero to one.
//
//sp:name GetCurrentCharge
func GetCurrentCharge(weapon int32) float32 {
	if !engine.HasEntProp(weapon, engine.PropSend(), "m_flChargeBeginTime") {
		return 0.0
	}

	charge := float32(0.0)

	chargeBeginTime := engine.EntPropFloat(weapon, engine.PropSend(), "m_flChargeBeginTime")

	if chargeBeginTime != 0.0 {
		charge = engine.MinFloat(1.0, engine.GameTime()-chargeBeginTime)
	}

	return charge
}

/*
TFBotNoticeThreat makes the game's own bot notice something now.

UpdateDelayedThreatNotices is called in CTFBotTacticalMonitor::Update, but that
behaviour can be interrupted, so it is called here to make sure he has noticed.
*/
//
//sp:name TFBot_NoticeThreat
func TFBotNoticeThreat(tfbot int32, threat int32) {
	engine.RunScriptCodeAt(tfbot, engine.Default(), engine.Default(),
		"self.DelayedThreatNotice(EntIndexToHScript(%d),0);self.UpdateDelayedThreatNotices()", threat)
}

/*
MovePlayerTowardsGoal is the WASD a bot is pushed with, as the two axes the game
reads rather than as a direction.

The forward vector is flattened and the goal is taken relative to it, so what comes
out is the four keys a person would be holding.
*/
//
//sp:name MovePlayerTowardsGoal
//sp:const vGoal
//sp:mutates vVel
func MovePlayerTowardsGoal(client int32, vGoal [3]float32, vVel [3]float32) {
	// WASD Movement
	forward3D := engine.EyeVectors(client)

	var vForward [3]float32

	vForward[0] = forward3D[0]
	vForward[1] = forward3D[1]

	_, vForward = engine.NormalizeVector(vForward)

	var right [3]float32

	right[0] = vForward[1]
	right[1] = -vForward[0]

	// PlayerLocomotion::GetFeet
	vFeet := engine.Origin(client)

	to := engine.SubtractVectors(vGoal, vFeet)

	_, to = engine.NormalizeVector(to)

	ahead := engine.VectorDotProduct(to, vForward)
	side := engine.VectorDotProduct(to, right)

	const epsilon = 0.25

	if ahead > epsilon {
		vVel[0] = engine.PlayerSideSpeed()
	} else if ahead < -epsilon {
		vVel[0] = -engine.PlayerSideSpeed()
	}

	if side <= -epsilon {
		vVel[1] = -engine.PlayerSideSpeed()
	} else if side >= epsilon {
		vVel[1] = engine.PlayerSideSpeed()
	}
}

// How fast an engineer walks to a build spot, and the clock either side of it.
const (
	//sp:name BUILD_WALK_SPEED
	walkSpeed = 180.0
	//sp:name BUILD_WALK_TIME_MIN
	walkTimeMin = 12.0
	//sp:name BUILD_WALK_TIME_MAX
	walkTimeMax = 40.0
)

// BuildReachTime is how long a walk to a build spot is worth waiting for, priced
// by its length rather than by a flat clock.
//
//sp:name BuildReachTime
//sp:const from
//sp:const to
func BuildReachTime(from [3]float32, to [3]float32) float32 {
	seconds := walkTimeMin + engine.VectorDistance(from, to)/walkSpeed

	return engine.ChooseFloat(seconds > walkTimeMax, walkTimeMax, seconds)
}
