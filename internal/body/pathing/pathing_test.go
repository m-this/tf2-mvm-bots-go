package pathing

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
A route that cannot be refused still spends the frame's budget, so the refresh
that can wait does (mvm-qk6). Before, the engineer's routes out of spawn and
IsPathToVectorPossible were searches PATHS_PER_FRAME never saw, and a frame
could hold them and two refreshes besides.
*/
func TestARouteThatCannotBeRefusedSpendsTheFrame(t *testing.T) {
	tick := int32(100)
	t.Cleanup(engine.InstallPathings(engine.PathingCalls{
		GameTickCount: func() int32 { return tick },
	}))
	t.Cleanup(func() { pathBudgetTick, pathsThisTick = 0, 0 })

	for range PathsPerFrame {
		SpendPathBudget()
	}
	if TakePathBudget() {
		t.Fatal("the refresh was given room in a frame the one-shot routes had already spent")
	}

	tick++
	if !TakePathBudget() {
		t.Fatal("the next frame has no room either; spending is per frame")
	}
}
