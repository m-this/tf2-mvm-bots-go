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
