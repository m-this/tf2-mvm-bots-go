/*
Package handle is the lifetime shape.

Every way out of Search has to delete the collector, and there are four: the two
returns inside the loop, the one after it, and falling off the end of Missing.
The golden file is what says the generator found all of them.
*/
package handle

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Search walks the ground near a point and returns the first area that is not a
// spawn room, leaving through three different returns on the way.
func Search(origin [3]float32, actor int32) (found [3]float32) {
	areas := engine.CollectAreasInRadius(origin, 300.0)
	defer areas.Close()

	for i := int32(0); i < areas.Count(); i++ {
		area := areas.Get(i)

		if area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}
		if area.HasAttributeTF(engine.BlueSpawnRoom()) {
			return found
		}

		centre := area.Center()

		if engine.IsPathToVectorPossible(actor, centre) {
			found = centre
			return found
		}
	}

	return found
}

// Missing falls off the end, which is a way out too.
func Missing(origin [3]float32) int32 {
	areas := engine.CollectAreasInRadius(origin, 100.0)
	defer areas.Close()

	count := areas.Count()

	return count
}

/*
Reload closes by hand rather than by defer, which is the shape that crashed.

A closed handle keeps the number it had, and SourceMod hands that number out
again. The variable then reads as neither null nor its own, so the guard that
asks whether it is null lets the next call through and the call throws. delete
is SourcePawn's close that writes null over the variable, so the golden pins
that a close written by hand comes out as delete and not as a call to Close.
*/
func Reload(origin [3]float32) int32 {
	kept.Close()
	kept = engine.CollectAreasInRadius(origin, 100.0)

	return kept.Count()
}

// kept outlives the function that fills it, which is why closing it by hand is
// the only way to close it at all.
//
//sp:writable
var kept engine.Areas
