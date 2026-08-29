package runmap

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
)

var update = flag.Bool("update", false, "rewrite the golden picture")

/*
A hand-built mesh rather than a shipped map.

The real meshes are a thousand areas each and a golden over one of them would
be a file nobody can read a diff of. Four areas in a known square is a picture
a person can check by looking at it, and it still exercises the projection, the
Y flip and the clipping.
*/
func fixtureMesh() *navmesh.Mesh {
	area := func(id uint32, x0, y0, x1, y1 float32) *navmesh.Area {
		return &navmesh.Area{
			ID:        navmesh.AreaID(id),
			NorthWest: navmesh.Vec3{X: x0, Y: y1, Z: 0},
			SouthEast: navmesh.Vec3{X: x1, Y: y0, Z: 0},
		}
	}

	return &navmesh.Mesh{Areas: []*navmesh.Area{
		area(1, 0, 0, 400, 400),
		area(2, 400, 0, 800, 400),
		area(3, 0, 400, 400, 800),
		area(4, 400, 400, 800, 800),
	}}
}

func fixtureWave() Wave {
	return Wave{
		Map:    "fixture",
		Number: 1,
		Tracks: []Track{
			{Who: "Waldo", Class: "engineer", Samples: []Sample{
				{At: []float64{50, 50, 0}},
				{At: []float64{350, 250, 0}},
				{At: []float64{700, 700, 0}},
			}},
			{Who: "Ada", Class: "medic", Samples: []Sample{
				{At: []float64{700, 100, 0}},
				{At: []float64{100, 700, 0}},
			}},
		},
		Buildings: []Building{
			{Owner: "Waldo", Type: "sentry", Level: 3, At: []float64{400, 400, 0}},
		},
	}
}

// The golden is small and deterministic: same mesh, same samples, same pixels.
// A change to the projection or the palette shows up here as a diff and has to
// be looked at rather than noticed later on a real map.
func TestDrawMatchesTheGolden(t *testing.T) {
	img := Draw(fixtureMesh(), fixtureWave(), 320)

	var got bytes.Buffer
	if err := PNG(&got, img); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("testdata", "fixture-wave1.png")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./internal/runmap -update to write it)", err)
	}

	if !bytes.Equal(got.Bytes(), want) {
		t.Errorf("the drawing changed; look at it, then rerun with -update if it is right")
	}
}

// The picture is the map's shape, not the frame's: a map twice as wide as it is
// deep must not be stretched square.
func TestThePictureKeepsTheMapsProportions(t *testing.T) {
	mesh := &navmesh.Mesh{Areas: []*navmesh.Area{{
		NorthWest: navmesh.Vec3{X: 0, Y: 400},
		SouthEast: navmesh.Vec3{X: 800, Y: 0},
	}}}

	bounds := Draw(mesh, Wave{}, 320).Bounds()

	width := bounds.Dx() - 2*margin
	height := bounds.Dy() - 2*margin

	if width != 2*height {
		t.Errorf("drew %dx%d inside the margins, want twice as wide as deep", width, height)
	}
}

// North is up. Getting this backwards produces a picture that looks fine and is
// mirrored, which is the kind of wrong that survives a long time.
func TestNorthIsUp(t *testing.T) {
	proj, _ := project(fixtureMesh(), 320)

	_, low := proj.at(0, 0)
	_, high := proj.at(0, 800)

	if high >= low {
		t.Errorf("y=800 drew at row %d and y=0 at row %d, want the larger world y higher up", high, low)
	}
}

// A sample outside the mesh is a real case: a bot pushed out of bounds, or a
// mesh that does not cover where he stands. It must not panic.
func TestASampleOffTheMeshIsClipped(t *testing.T) {
	wave := Wave{Tracks: []Track{{Who: "Stray", Class: "spy", Samples: []Sample{
		{At: []float64{-90000, -90000, 0}},
		{At: []float64{90000, 90000, 0}},
	}}}}

	img := Draw(fixtureMesh(), wave, 320)

	if img.Bounds().Empty() {
		t.Error("drew nothing at all")
	}
}

// An unknown class still has to be drawn, since a picture missing a bot is
// worse than a picture with a grey one.
func TestAnUnknownClassGetsAColourAnyway(t *testing.T) {
	colour, known := ClassColour("archimedes")

	if known {
		t.Error("archimedes is not a class this package knows")
	}

	if colour.A != 0xFF {
		t.Errorf("colour %v is transparent, so nothing would be drawn", colour)
	}
}
