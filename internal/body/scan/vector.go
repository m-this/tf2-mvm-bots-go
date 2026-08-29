package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
The two vector helpers every scan above measures with.

Both are three lines in util.sp and both are the float[] form: the plugin's own
callers use them inline as arguments to GetVectorDistance, so they are emitted
that way rather than as the parameter form the rest of the generated code uses.
Changing the shape would rewrite call sites this port has not reached.

They were externs while the scans were being ported. They are bodies now, and
the externs that named them are gone: internal/body refuses to have both.
*/

// WorldSpaceCenter is util.sp:348. The middle of the entity, which is what the
// plugin measures ranges from rather than the feet.
//
//sp:returns
//sp:name WorldSpaceCenter
func WorldSpaceCenter(entity int32) (centre [3]float32) {
	centre = engine.EntityWorldSpaceCenter(entity)
	return centre
}

// AbsOrigin is util.sp:542. Where the entity is.
//
//sp:returns
//sp:name GetAbsOrigin
func AbsOrigin(entity int32) (origin [3]float32) {
	origin = engine.EntityAbsOrigin(entity)
	return origin
}
