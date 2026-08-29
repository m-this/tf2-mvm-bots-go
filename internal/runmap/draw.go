package runmap

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"sort"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
)

// DefaultSize is the longest side of the picture in pixels. The shorter side
// follows the map's own proportions, so a map is never stretched to fill a
// frame.
const DefaultSize = 1600

// Margin keeps the outermost nav areas off the edge, where a line drawn one
// pixel outside the image is a line nobody sees.
const margin = 16

/*
Palette is one colour per class, and grey for anything unnamed.

Chosen so that the six seats of a lineup stay apart at one pixel wide, which
rules out the game's own red: every track would be a shade of it. The mesh
underneath is drawn in greys so that every coloured pixel is the run and not
the map.
*/
var palette = map[string]color.RGBA{
	"scout":    {0xF2, 0xC5, 0x4E, 0xFF},
	"soldier":  {0xE2, 0x6A, 0x2C, 0xFF},
	"pyro":     {0xE0, 0x5C, 0x8A, 0xFF},
	"demoman":  {0x7A, 0x4E, 0x2A, 0xFF},
	"heavy":    {0x9B, 0x59, 0xD0, 0xFF},
	"engineer": {0x2E, 0x8B, 0xC0, 0xFF},
	"medic":    {0x3F, 0xB9, 0x50, 0xFF},
	"sniper":   {0x1F, 0x6F, 0x4A, 0xFF},
	"spy":      {0xC0, 0x39, 0x39, 0xFF},
}

var (
	colourUnknown  = color.RGBA{0x88, 0x88, 0x88, 0xFF}
	colourArea     = color.RGBA{0x27, 0x2B, 0x30, 0xFF}
	colourAreaEdge = color.RGBA{0x3A, 0x40, 0x47, 0xFF}
	colourVoid     = color.RGBA{0x14, 0x16, 0x19, 0xFF}
	colourBuilding = color.RGBA{0xEE, 0xEE, 0xEE, 0xFF}
)

// ClassColour is the colour a class is drawn in, and whether it is a class this
// package knows. Exported so the caller can print a legend, since the picture
// carries no text.
func ClassColour(class string) (color.RGBA, bool) {
	c, ok := palette[class]
	if !ok {
		return colourUnknown, false
	}

	return c, true
}

// Classes lists the classes in a wave, in the order a legend should print them.
func (w Wave) Classes() []string {
	seen := map[string]bool{}
	var classes []string

	for _, track := range w.Tracks {
		if track.Class != "" && !seen[track.Class] {
			seen[track.Class] = true
			classes = append(classes, track.Class)
		}
	}

	sort.Strings(classes)

	return classes
}

/*
projection turns world coordinates into pixels.

Only X and Y. A nav mesh is a floor plan and the third axis is what makes a
floor plan hard to read: two areas stacked over each other are the same square
from above, which is what a plan is for. Height is where a picture stops being
the right tool and the numbers take over.

Y is flipped because the world counts north as increasing and an image counts
down.
*/
type projection struct {
	minX, minY float64
	scale      float64
	height     int
}

func project(mesh *navmesh.Mesh, size int) (projection, image.Rectangle) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)

	for _, area := range mesh.Areas {
		minX = math.Min(minX, math.Min(float64(area.NorthWest.X), float64(area.SouthEast.X)))
		maxX = math.Max(maxX, math.Max(float64(area.NorthWest.X), float64(area.SouthEast.X)))
		minY = math.Min(minY, math.Min(float64(area.NorthWest.Y), float64(area.SouthEast.Y)))
		maxY = math.Max(maxY, math.Max(float64(area.NorthWest.Y), float64(area.SouthEast.Y)))
	}

	// A mesh with no areas would divide by zero below. It is not a picture
	// worth drawing, but it should not be a panic either.
	spanX, spanY := maxX-minX, maxY-minY
	if !(spanX > 0) || !(spanY > 0) {
		return projection{height: 1}, image.Rect(0, 0, 1, 1)
	}

	usable := float64(size - 2*margin)
	scale := math.Min(usable/spanX, usable/spanY)

	width := int(spanX*scale) + 2*margin
	height := int(spanY*scale) + 2*margin

	return projection{minX: minX, minY: minY, scale: scale, height: height},
		image.Rect(0, 0, width, height)
}

func (p projection) at(x, y float64) (int, int) {
	px := int((x-p.minX)*p.scale) + margin

	// The flip is here and nowhere else, so there is one place to be wrong.
	py := p.height - margin - int((y-p.minY)*p.scale)

	return px, py
}

/*
Draw renders one wave over the mesh it was played on.

The order is deliberate: the mesh first, then buildings, then the bots on top.
A bot standing on his own sentry is the interesting case and he should not be
hidden by it.
*/
func Draw(mesh *navmesh.Mesh, wave Wave, size int) *image.RGBA {
	if size <= 4*margin {
		size = DefaultSize
	}

	proj, bounds := project(mesh, size)
	img := image.NewRGBA(bounds)

	fill(img, colourVoid)

	for _, area := range mesh.Areas {
		x0, y0 := proj.at(float64(area.NorthWest.X), float64(area.NorthWest.Y))
		x1, y1 := proj.at(float64(area.SouthEast.X), float64(area.SouthEast.Y))
		rectangle(img, x0, y0, x1, y1, colourArea, colourAreaEdge)
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

/*
MaxGroundSpeed is the fastest a defender crosses ground, which is the Scout at
400 units a second. Used to ask whether two samples could be the same walk.
*/
const MaxGroundSpeed = 400.0

/*
drawTrack draws where a bot was, and joins two samples only when the join could
have happened.

Drawn as dots first, because the telemetry samples every five seconds and a
straight line between two of those is an invention: four hundred units a second
is two thousand units of unseen walking per sample, most of the width of a map.
The first version joined every pair and produced a cat's cradle across Decoy
that showed nothing except that six bots had been busy.

Dots do not lie the same way. Where a bot spent time, its samples pile up in one
place and read as a bright knot, which is exactly the shape of the faults worth
finding: a bot wedged in a corner, a team that never left spawn, a nest nobody
moved off. Time spent is what the sampling can honestly show.

The line stays for the case where it is true. Two samples close enough together
that the bot could have walked between them are one movement, and joining those
says which way he was going.
*/
func drawTrack(img *image.RGBA, proj projection, track Track, colour color.RGBA) {
	var (
		last         Sample
		lastX, lastY int
		started      bool
	)

	for _, s := range track.Samples {
		x, y := proj.at(s.At[0], s.At[1])

		if started && couldWalk(last, s) {
			line(img, lastX, lastY, x, y, colour)
		}

		dot(img, x, y, colour)

		last, lastX, lastY, started = s, x, y, true
	}

	// Where he ended up, marked apart from the dots so the end of a track is
	// findable in a knot of them.
	if started {
		marker(img, lastX, lastY, colour)
	}
}

/*
JoinInterval is the longest gap between two samples that is still worth drawing
a line across.

One second, which the current telemetry never satisfies: it samples every five,
and this is deliberately set below that so no line is drawn from it at all.

The first cut allowed anything the bot could physically have walked, which at
five seconds is two thousand units and most of Decoy, so essentially every pair
passed and the picture stayed a cat's cradle. The question a line answers is not
"could he have got there" but "do I know how he got there", and at five seconds
the answer is no however fast he runs.

It is a threshold rather than a flag so that a finer sampler gets its lines for
free. The break-time sampler runs at a quarter second and would draw a real
path.
*/
const JoinInterval = 1.0

/*
couldWalk asks whether one sample is the other one walked to.

Two tests, and both have to pass. Close enough in time that the sampling saw the
movement, and close enough in distance that a defender could have covered it. A
pair that fails the second is a bot that was carried, teleported, or killed and
respawned, and none of those is a line.

A pair with no time between them fails: an unset clock would otherwise make
every jump look instantaneous and therefore plausible.
*/
func couldWalk(from, to Sample) bool {
	elapsed := to.Clock - from.Clock
	if elapsed <= 0 || elapsed > JoinInterval {
		return false
	}

	dx := to.At[0] - from.At[0]
	dy := to.At[1] - from.At[1]

	return math.Hypot(dx, dy) <= elapsed*MaxGroundSpeed
}

// dot is one sample. Small, because the picture's density is the measurement:
// a fat dot fills a room with one visit and says the bot lived there.
func dot(img *image.RGBA, x, y int, c color.RGBA) {
	set(img, x, y, c)
	set(img, x+1, y, c)
	set(img, x, y+1, c)
	set(img, x+1, y+1, c)
}

// PNG writes an image out.
func PNG(w io.Writer, img *image.RGBA) error {
	return png.Encode(w, img)
}

func fill(img *image.RGBA, c color.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// rectangle fills a nav area and outlines it, so two areas that touch are still
// two areas rather than one blob.
func rectangle(img *image.RGBA, x0, y0, x1, y1 int, fill, edge color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			onEdge := x == x0 || x == x1 || y == y0 || y == y1
			if onEdge {
				set(img, x, y, edge)
			} else {
				set(img, x, y, fill)
			}
		}
	}
}

// marker is a small cross rather than a dot: one pixel disappears against the
// mesh edges, and a filled square hides what it sits on.
func marker(img *image.RGBA, x, y int, c color.RGBA) {
	const arm = 3

	for d := -arm; d <= arm; d++ {
		set(img, x+d, y, c)
		set(img, x, y+d, c)
	}
}

// line is Bresenham. Integer only, so a track drawn twice is drawn the same.
func line(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)

	stepX, stepY := 1, 1
	if x0 > x1 {
		stepX = -1
	}
	if y0 > y1 {
		stepY = -1
	}

	err := dx + dy

	// Bounded by the diagonal of the image: the loop below terminates when it
	// reaches the far point, and this is the backstop if it ever cannot.
	limit := dx - dy + 2

	for range limit {
		set(img, x0, y0, c)

		if x0 == x1 && y0 == y1 {
			return
		}

		doubled := 2 * err
		if doubled >= dy {
			err += dy
			x0 += stepX
		}
		if doubled <= dx {
			err += dx
			y0 += stepY
		}
	}
}

// set is the only writer, so everything this package draws is clipped in one
// place rather than each caller checking the bounds it happens to remember.
func set(img *image.RGBA, x, y int, c color.RGBA) {
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		return
	}

	img.SetRGBA(x, y, c)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
