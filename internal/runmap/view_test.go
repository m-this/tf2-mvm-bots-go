package runmap

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
)

// A window is the map seen up close: the same world point lands in the same
// place relative to the window's corners, whatever the mesh around it spans.
func TestAWindowFramesItsOwnCorners(t *testing.T) {
	view := View{Size: 320, Window: Window{MinX: 100, MinY: 100, MaxX: 300, MaxY: 300}}

	proj, bounds := projectWindow(fixtureMesh(), view.Window, view.Size)

	x, y := proj.at(100, 100)
	if x != margin || y != bounds.Dy()-margin {
		t.Errorf("the south-west corner drew at %d,%d, want %d,%d", x, y, margin, bounds.Dy()-margin)
	}

	x, y = proj.at(300, 300)
	if x != bounds.Dx()-margin || y != margin {
		t.Errorf("the north-east corner drew at %d,%d, want %d,%d", x, y, bounds.Dx()-margin, margin)
	}
}

// An empty window is the whole mesh, so the old call and the new one agree.
func TestAnEmptyWindowIsTheWholeMesh(t *testing.T) {
	whole := Draw(fixtureMesh(), fixtureWave(), 320)
	viewed := DrawView(fixtureMesh(), fixtureWave(), View{Size: 320})

	if whole.Bounds() != viewed.Bounds() {
		t.Fatalf("bounds differ: %v against %v", whole.Bounds(), viewed.Bounds())
	}

	for i := range whole.Pix {
		if whole.Pix[i] != viewed.Pix[i] {
			t.Fatalf("pixel %d differs", i)
		}
	}
}

// A spot is drawn in its kind's colour, at its own place, and a fall in red.
func TestSpotsAndFallsAreDrawnWhereTheyAre(t *testing.T) {
	view := View{
		Size:  320,
		Spots: []navmesh.Spot{{Kind: navmesh.TeleporterExit, Origin: navmesh.Vec3{X: 200, Y: 200}}},
		Falls: []navmesh.Fall{{At: navmesh.Vec3{X: 600, Y: 600}, Descent: navmesh.FallLethalHeight}},
	}

	img := DrawView(fixtureMesh(), Wave{}, view)
	proj, _ := project(fixtureMesh(), view.Size)

	x, y := proj.at(200, 200)
	if got := img.RGBAAt(x+7, y); got != SpotColour(navmesh.TeleporterExit) {
		t.Errorf("the exit's diamond tip is %v, want %v", got, SpotColour(navmesh.TeleporterExit))
	}

	x, y = proj.at(600, 600)
	if got := img.RGBAAt(x, y); got != colourFallKills {
		t.Errorf("the fall drew as %v, want %v", got, colourFallKills)
	}
}
