/*
Package angles is the part of source/redbots3/util.sp that turns one direction
into another: the fold to the shortest way round, and the step towards a target.
*/
package angles

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// AngleNormalize folds an angle into the half turn either side of zero.
//
//sp:name AngleNormalize
func AngleNormalize(angle float32) float32 {
	//nolint:gocritic // assignOp: the shipped file writes the subtraction out, and the port keeps its shape
	angle = angle - 360.0*float32(engine.RoundToFloor(angle/360.0))

	for angle > 180.0 {
		angle -= 360.0
	}

	for angle < -180.0 {
		angle += 360.0
	}

	return angle
}

// AngleDiff is how far one angle is from another, the short way.
//
//sp:name AngleDiff
func AngleDiff(destAngle float32, srcAngle float32) float32 {
	return AngleNormalize(destAngle - srcAngle)
}

// ApproachAngle is one step of that difference, capped at a speed.
//
//sp:name ApproachAngle
func ApproachAngle(target float32, value float32, speed float32) float32 {
	delta := AngleDiff(target, value)

	if speed < 0.0 {
		speed = -speed
	}

	//nolint:gocritic // ifElseChain: the shipped file is this chain, and the port keeps its shape
	if delta > speed {
		value += speed
	} else if delta < -speed {
		value -= speed
	} else {
		value = target
	}

	return AngleNormalize(value)
}

// GetAbsVelocity is how fast the entity is going. Its SourcePawn returns the
// array.
//
//sp:name GetAbsVelocity
//sp:returns
func GetAbsVelocity(entity int32) (vec [3]float32) {
	vec = engine.EntityOf(entity).AbsVelocity()

	return vec
}

// GetEyePosition is where the client looks from. Its SourcePawn returns the
// array.
//
//sp:name GetEyePosition
//sp:returns
func GetEyePosition(client int32) (vec [3]float32) {
	vec = engine.EyePositionOf(client)

	return vec
}

/*
VectorNormalize makes a vector unit length and answers how long it was, by the
inverse square root the engine itself uses.

Kept as the engine writes it rather than as NormalizeVector, because the two
disagree in the last places and this one is what the aim was measured with.

The vector is scaled in place, which is what the shipped file does and what its
callers rely on. SourcePawn passes the array by reference and Go copies it, so the
Go here does not describe what the emitted function does to its caller: that is
what //sp:mutates says.
*/
//
//sp:name VMX_VectorNormalize
//sp:mutates a1
func VectorNormalize(a1 [3]float32) float32 {
	length := engine.VectorLengthSquared(a1, true) + 0.0000000001
	v4 := 1.0 / engine.SquareRoot(length)
	den := v4 * ((3.0 - ((v4 * v4) * length)) * 0.5)

	//nolint:ineffassign,wastedassign,staticcheck // the write is the point: //sp:mutates says the emitted ScaleVector scales the caller's array
	a1 = engine.ScaleVector(a1, den)

	return den * length
}
