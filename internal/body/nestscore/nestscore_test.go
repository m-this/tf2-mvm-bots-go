package nestscore

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// Some community maps have a loaded nav mesh but no areas in range of the
// population file's bomb target. CBaseNPC reports that as a null collector,
// which is an ordinary empty sample rather than a handle Count may inspect.
func TestCollectBombApproachAreasAcceptsANullCollector(t *testing.T) {
	restore := engine.InstallNav(engine.NavCalls{
		CollectAreasInRadius: func([3]float32, float32) engine.Areas {
			return engine.NoAreas()
		},
		AreasCount: func(engine.Areas) int32 {
			t.Fatal("Count called on a null area collector")
			return 0
		},
		AreasClose: func(engine.Areas) {
			t.Fatal("Close called on a null area collector")
		},
	})
	defer restore()

	CollectBombApproachAreas([3]float32{}, 1300.0, engine.NoList())
}
