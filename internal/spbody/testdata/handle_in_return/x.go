package x

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Count reads the handle in the expression that returns. Go computes the result
// and then runs the defer; SourcePawn would delete first and read after.
func Count(origin [3]float32) int32 {
	areas := engine.CollectAreasInRadius(origin, 300.0)
	defer areas.Close()

	return areas.Count()
}
