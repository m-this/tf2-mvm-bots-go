package x

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Count opens a collector and never closes it, which is a leak every frame this
// runs on.
func Count(origin [3]float32) int32 {
	areas := engine.CollectAreasInRadius(origin, 300.0)

	return areas.Count()
}
