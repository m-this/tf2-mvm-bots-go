package runmap

import (
	"image"
	"image/color"
	"math"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
)

/*
View is what to draw a wave through: how large, which part of the map, and
what to lay over it besides the run.

The zero Window is the whole mesh. A run drawn at 1600 pixels across Bigrock
puts a nest and the rock beside it two pixels apart, so a question about one
spot is asked through a window a few hundred units wide.
*/
type View struct {
	Size   int
	Window Window
	Spots  []navmesh.Spot
	Falls  []navmesh.Fall
}

// Window is a rectangle of the world, in map units. Empty means everything.
type Window struct {
	MinX, MinY, MaxX, MaxY float64
}

func (w Window) empty() bool { return w.MaxX <= w.MinX || w.MaxY <= w.MinY }

// HurtingFalls lists every fall on the mesh that costs health, once per edge,
// which is what a picture of where not to build wants.
func HurtingFalls(mesh *navmesh.Mesh) []navmesh.Fall {
	var out []navmesh.Fall

	for _, area := range mesh.Areas {
		for _, f := range mesh.Falls(area.ID) {
			if f.Descent >= navmesh.FallDamageHeight {
				out = append(out, f)
			}
		}
	}

	return out
}

var spotColours = map[navmesh.SpotKind]color.RGBA{
	navmesh.EngineerNest:   {0xFF, 0x3C, 0xE0, 0xFF},
	navmesh.NestNoTank:     {0xFF, 0x8C, 0xE8, 0xFF},
	navmesh.NestTankOnly:   {0xB0, 0x20, 0xA0, 0xFF},
	navmesh.TeleporterExit: {0x00, 0xE5, 0xFF, 0xFF},
	navmesh.DispenserSpot:  {0xC8, 0xFF, 0x40, 0xFF},
	navmesh.SniperSpot:     {0x80, 0xA0, 0xFF, 0xFF},
}

var (
	colourFallHurts = color.RGBA{0xFF, 0x40, 0x40, 0xFF}
	colourFallKills = color.RGBA{0xFF, 0x00, 0x00, 0xFF}
)

// SpotColour is the colour a spot kind is drawn in, for the legend.
func SpotColour(kind navmesh.SpotKind) color.RGBA {
	if c, ok := spotColours[kind]; ok {
		return c
	}

	return colourUnknown
}

/*
DrawView renders one wave through a view.

The order is deliberate: the mesh, the falls, the spots, the buildings, then
the bots on top. A bot standing on his own sentry is the interesting case and
he should not be hidden by it.
*/
func DrawView(mesh *navmesh.Mesh, wave Wave, view View) *image.RGBA {
	size := view.Size
	if size <= 4*margin {
		size = DefaultSize
	}

	proj, bounds := projectWindow(mesh, view.Window, size)
	img := image.NewRGBA(bounds)

	fill(img, colourVoid)

	for _, area := range mesh.Areas {
		x0, y0 := proj.at(float64(area.NorthWest.X), float64(area.NorthWest.Y))
		x1, y1 := proj.at(float64(area.SouthEast.X), float64(area.SouthEast.Y))
		rectangle(img, x0, y0, x1, y1, colourArea, colourAreaEdge)
	}

	for _, f := range view.Falls {
		c := colourFallHurts
		if f.Descent >= navmesh.FallLethalHeight {
			c = colourFallKills
		}
		x, y := proj.at(float64(f.At.X), float64(f.At.Y))
		dot(img, x, y, c)
	}

	for _, s := range view.Spots {
		x, y := proj.at(float64(s.Origin.X), float64(s.Origin.Y))
		diamond(img, x, y, SpotColour(s.Kind))
	}

	for _, b := range wave.Buildings {
		x, y := proj.at(b.At[0], b.At[1])
		marker(img, x, y, colourBuilding)
	}

	for _, track := range wave.Tracks {
		colour, _ := ClassColour(track.Class)
		drawTrack(img, proj, track, colour)
	}

	return img
}

// projectWindow is project over a window instead of the whole mesh. Areas
// outside it still get drawn and land off the image, where set clips them.
func projectWindow(mesh *navmesh.Mesh, w Window, size int) (projection, image.Rectangle) {
	if w.empty() {
		return project(mesh, size)
	}

	spanX, spanY := w.MaxX-w.MinX, w.MaxY-w.MinY
	usable := float64(size - 2*margin)
	scale := math.Min(usable/spanX, usable/spanY)

	width := int(spanX*scale) + 2*margin
	height := int(spanY*scale) + 2*margin

	return projection{minX: w.MinX, minY: w.MinY, scale: scale, height: height},
		image.Rect(0, 0, width, height)
}

// diamond is a hollow lozenge, so a spot reads apart from a building's cross
// and from a bot's dots, and what it sits on stays visible.
func diamond(img *image.RGBA, x, y int, c color.RGBA) {
	const arm = 7

	for d := 0; d <= arm; d++ {
		set(img, x+d, y-(arm-d), c)
		set(img, x+d, y+(arm-d), c)
		set(img, x-d, y-(arm-d), c)
		set(img, x-d, y+(arm-d), c)
	}
}
