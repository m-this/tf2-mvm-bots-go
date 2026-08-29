package navmesh

import "math"

// gridCell is how wide one bucket of the lookup grid is. The engine uses the
// same idea with the same order of size, and 300 units puts a handful of areas
// in a cell on every shipped map without making the grid itself large.
const gridCell float32 = 300

// grid buckets areas by their footprint so a query about a position looks at the
// areas near it rather than at all of them.
//
// Without it every query is a scan of the whole mesh, and the sweeps in the
// report do a query per side per area: measured, that was a minute of a test
// run on one package. Areas overlap and a footprint spans several cells, so an
// area appears in every cell it touches and a caller has to expect the same area
// twice only if it does not go through these methods.
type grid struct {
	minX, minY float32
	cellsX     int
	cellsY     int
	cells      [][]*Area
}

func newGrid(areas []*Area) *grid {
	if len(areas) == 0 {
		return &grid{cellsX: 1, cellsY: 1, cells: make([][]*Area, 1)}
	}

	g := &grid{
		minX: float32(math.Inf(1)),
		minY: float32(math.Inf(1)),
	}
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))

	for _, a := range areas {
		g.minX = minf(g.minX, a.NorthWest.X)
		g.minY = minf(g.minY, a.NorthWest.Y)
		maxX = maxf(maxX, a.SouthEast.X)
		maxY = maxf(maxY, a.SouthEast.Y)
	}

	g.cellsX = int((maxX-g.minX)/gridCell) + 1
	g.cellsY = int((maxY-g.minY)/gridCell) + 1
	g.cells = make([][]*Area, g.cellsX*g.cellsY)

	for _, a := range areas {
		loX, loY := g.cellOf(a.NorthWest.X, a.NorthWest.Y)
		hiX, hiY := g.cellOf(a.SouthEast.X, a.SouthEast.Y)
		for y := loY; y <= hiY; y++ {
			for x := loX; x <= hiX; x++ {
				i := y*g.cellsX + x
				g.cells[i] = append(g.cells[i], a)
			}
		}
	}

	return g
}

func (g *grid) cellOf(x, y float32) (int, int) {
	cx := clampi(int((x-g.minX)/gridCell), 0, g.cellsX-1)
	cy := clampi(int((y-g.minY)/gridCell), 0, g.cellsY-1)
	return cx, cy
}

// near calls f for every area whose footprint is within radius of (x, y),
// possibly more than once for an area spanning several cells.
func (g *grid) near(x, y, radius float32, f func(*Area)) {
	loX, loY := g.cellOf(x-radius, y-radius)
	hiX, hiY := g.cellOf(x+radius, y+radius)

	for cy := loY; cy <= hiY; cy++ {
		for cx := loX; cx <= hiX; cx++ {
			for _, a := range g.cells[cy*g.cellsX+cx] {
				f(a)
			}
		}
	}
}

// at calls f for every area whose footprint could contain (x, y).
func (g *grid) at(x, y float32, f func(*Area)) {
	cx, cy := g.cellOf(x, y)
	for _, a := range g.cells[cy*g.cellsX+cx] {
		f(a)
	}
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
