package navmesh

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

// navMagic is the word every .nav file opens with, CNavMesh::NAV_MAGIC_NUMBER.
const navMagic uint32 = 0xFEEDFACE

// The one format this package decodes. Team Fortress 2 ships version 16
// subversion 2 for every map in tf2_misc_dir.vpk, and a file outside that pair
// is refused by version number rather than guessed at, because the layout moved
// at 8, 11, 13, 14, 15 and 16 and a wrong guess reads plausible garbage.
const (
	SupportedVersion    uint32 = 16
	SupportedSubVersion uint32 = 2
)

// ErrNotNav is returned for a file that does not open with the nav magic word.
var ErrNotNav = errors.New("navmesh: not a nav file")

// ErrUnsupportedVersion is returned for a nav file this package will not decode.
// The message carries the version and subversion so an unreadable map is a
// number and not a shrug.
var ErrUnsupportedVersion = errors.New("navmesh: unsupported nav version")

// LoadFile reads a .nav file from disk. A path ending in .gz is decompressed
// first, which is how the test data is stored: the shipped meshes are a megabyte
// each and compress to a quarter of that.
func LoadFile(path string) (*Mesh, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("navmesh: reading %s: %w", path, err)
	}

	if len(raw) > 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("navmesh: %s: %w", path, err)
		}
		defer func() { _ = zr.Close() }()

		raw, err = io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("navmesh: %s: %w", path, err)
		}
	}

	m, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("navmesh: %s: %w", path, err)
	}

	return m, nil
}

// Parse decodes a whole nav file.
//
// The file is trusted to be exactly as long as its contents say. Every count in
// it is read before the run it governs, so a truncated or misread file runs off
// the end and is reported as such rather than producing a shorter mesh; and the
// parse insists it finished on the last byte, which is the one check that
// catches a layout misread that happens to stay in bounds.
func Parse(data []byte) (*Mesh, error) {
	r := &reader{data: data}

	magic := r.u32()
	if r.err != nil || magic != navMagic {
		return nil, ErrNotNav
	}

	m := &Mesh{}
	m.Version = r.u32()
	m.SubVersion = r.u32()
	m.BSPSize = r.u32()
	m.IsAnalyzed = r.u8() != 0

	if r.err != nil {
		return nil, r.err
	}
	if m.Version != SupportedVersion || m.SubVersion != SupportedSubVersion {
		return nil, fmt.Errorf("%w: version %d subversion %d", ErrUnsupportedVersion, m.Version, m.SubVersion)
	}

	placeCount := int(r.u16())
	m.Places = make([]string, 0, placeCount)
	for range placeCount {
		m.Places = append(m.Places, r.lenString())
	}
	r.u8() // hasUnnamedAreas, of no use to a reader that never edits the mesh

	areaCount := int(r.u32())
	if r.err != nil {
		return nil, r.err
	}
	if areaCount > maxAreas {
		return nil, fmt.Errorf("navmesh: area count %d over the %d this reader accepts", areaCount, maxAreas)
	}

	m.Areas = make([]*Area, 0, areaCount)
	for i := range areaCount {
		a, err := readArea(r)
		if err != nil {
			return nil, fmt.Errorf("navmesh: area %d of %d: %w", i, areaCount, err)
		}
		m.Areas = append(m.Areas, a)
	}

	// Ladders close the file. Nothing here walks one, so they are counted and
	// stepped over; the count still has to be right for the end-of-file check.
	ladderCount := int(r.u32())
	if r.err != nil {
		return nil, r.err
	}
	if ladderCount > maxLadders {
		return nil, fmt.Errorf("navmesh: ladder count %d over the %d this reader accepts", ladderCount, maxLadders)
	}
	for range ladderCount {
		r.skip(ladderBytes)
	}

	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(data) {
		return nil, fmt.Errorf("navmesh: read %d of %d bytes, so the layout is wrong", r.off, len(data))
	}

	m.index()

	return m, nil
}

// Bounds a malformed or misread file cannot cross. The largest mesh Team
// Fortress 2 ships is 1328 areas, so these are three orders of margin and exist
// only so a wrong offset allocates nothing.
const (
	maxAreas       = 1 << 20
	maxLadders     = 1 << 16
	maxConnections = 1 << 16
	maxHidingSpots = 1 << 8
	maxEncounters  = 1 << 16
	maxVisible     = 1 << 20

	// id, four floats of top and bottom, width, direction, plus the four areas
	// at its ends.
	ladderBytes = 4 + 7*4 + 4 + 4*4
)

func readArea(r *reader) (*Area, error) {
	a := &Area{}
	a.ID = AreaID(r.u32())
	a.Attributes = Attributes(r.u32())
	a.NorthWest = r.vec3()
	a.SouthEast = r.vec3()
	a.NorthEastZ = r.f32()
	a.SouthWestZ = r.f32()

	for d := range NumDirections {
		n := int(r.u32())
		if r.err != nil {
			return nil, r.err
		}
		if n > maxConnections {
			return nil, fmt.Errorf("connection count %d in direction %d", n, d)
		}
		if n == 0 {
			continue
		}
		conns := make([]AreaID, 0, n)
		for range n {
			conns = append(conns, AreaID(r.u32()))
		}
		a.Connections[d] = conns
	}

	// Hiding spots: an id, a position and a flags byte each. Cover and sniper
	// hints are the bot's business, not this model's.
	hiding := int(r.u8())
	if hiding > maxHidingSpots {
		return nil, fmt.Errorf("hiding spot count %d", hiding)
	}
	r.skip(hiding * (4 + 12 + 1))

	// Encounter paths: a spot list per pair of ways through the area. Variable
	// length, so it is walked rather than skipped.
	encounters := int(r.u32())
	if r.err != nil {
		return nil, r.err
	}
	if encounters > maxEncounters {
		return nil, fmt.Errorf("encounter path count %d", encounters)
	}
	for range encounters {
		r.skip(4 + 1 + 4 + 1)
		spots := int(r.u8())
		r.skip(spots * (4 + 1))
		if r.err != nil {
			return nil, r.err
		}
	}

	a.Place = r.u16()

	// The ladders at the area's up and down ends.
	for range 2 {
		n := int(r.u32())
		if r.err != nil {
			return nil, r.err
		}
		if n > maxConnections {
			return nil, fmt.Errorf("ladder connection count %d", n)
		}
		r.skip(n * 4)
	}

	r.skip(2 * 4) // earliest occupy time per team
	r.skip(4 * 4) // light intensity per corner

	visible := int(r.i32())
	if r.err != nil {
		return nil, r.err
	}
	if visible < 0 || visible > maxVisible {
		return nil, fmt.Errorf("visible area count %d", visible)
	}
	if visible > 0 {
		a.Visible = make([]VisibleArea, 0, visible)
		for range visible {
			id := AreaID(r.u32())
			a.Visible = append(a.Visible, VisibleArea{ID: id, Visibility: Visibility(r.u8())})
		}
	}

	a.InheritVisibility = AreaID(r.u32())
	a.TFAttributes = TFAttributes(r.u32())

	return a, r.err
}

func (m *Mesh) index() {
	m.byID = make(map[AreaID]*Area, len(m.Areas))
	for _, a := range m.Areas {
		m.byID[a.ID] = a
	}

	m.grid = newGrid(m.Areas)

	m.incoming = make(map[AreaID][]AreaID, len(m.Areas))
	for _, a := range m.Areas {
		for d := range a.Connections {
			for _, id := range a.Connections[d] {
				m.incoming[id] = append(m.incoming[id], a.ID)
			}
		}
	}
}

// reader is a little-endian cursor over the file. It records the first failure
// and returns zeroes afterwards, so a run of reads is checked once at the end of
// the block that owns it rather than after every field.
type reader struct {
	data []byte
	off  int
	err  error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if r.off+n > len(r.data) {
		r.err = fmt.Errorf("navmesh: %w at offset %d wanting %d bytes", io.ErrUnexpectedEOF, r.off, n)
		return nil
	}
	b := r.data[r.off : r.off+n]
	r.off += n
	return b
}

func (r *reader) skip(n int) { r.take(n) }

func (r *reader) u8() uint8 {
	b := r.take(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (r *reader) u16() uint16 {
	b := r.take(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

// i32 reads a signed count. The file writes the visible-area count with
// CUtlBuffer::PutInt, so the four bytes are a signed int and reinterpreting them
// as one is the whole job; a negative result is a misread and the caller refuses
// it rather than allocating on it.
func (r *reader) i32() int32 {
	//nolint:gosec // G115: the bytes are a signed int in the file, so this is a reinterpretation and not a narrowing.
	return int32(r.u32())
}

func (r *reader) u32() uint32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *reader) f32() float32 {
	return math.Float32frombits(r.u32())
}

func (r *reader) vec3() Vec3 {
	return Vec3{r.f32(), r.f32(), r.f32()}
}

func (r *reader) lenString() string {
	n := int(r.u16())
	b := r.take(n)
	if b == nil {
		return ""
	}
	return string(b)
}
