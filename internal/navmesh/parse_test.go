package navmesh

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// TestParseShippedMaps is the parser's real test: the seven meshes Team
// Fortress 2 ships, read whole.
//
// Parse already insists it consumed the last byte of the file, which is the
// check that catches a layout misread that stays in bounds. What is added here
// is that the graph closes: every connection names an area that exists, so the
// area records cannot have been read at a drifting offset and still pass.
func TestParseShippedMaps(t *testing.T) {
	for _, name := range shippedMaps {
		t.Run(name, func(t *testing.T) {
			m := loadMap(t, name)

			if m.Version != SupportedVersion || m.SubVersion != SupportedSubVersion {
				t.Fatalf("version %d.%d, want %d.%d", m.Version, m.SubVersion, SupportedVersion, SupportedSubVersion)
			}
			if len(m.Areas) == 0 {
				t.Fatal("no areas")
			}

			seen := make(map[AreaID]bool, len(m.Areas))
			for _, a := range m.Areas {
				if seen[a.ID] {
					t.Fatalf("area %d appears twice", a.ID)
				}
				seen[a.ID] = true
			}

			for _, a := range m.Areas {
				for _, id := range a.Neighbours() {
					if m.Area(id) == nil {
						t.Fatalf("area %d connects to %d, which is not in the file", a.ID, id)
					}
				}
			}
		})
	}
}

// TestAreaGeometry checks the invariants every area in a real mesh holds, which
// is how a corner read in the wrong order would show.
func TestAreaGeometry(t *testing.T) {
	for _, name := range shippedMaps {
		t.Run(name, func(t *testing.T) {
			m := loadMap(t, name)

			for _, a := range m.Areas {
				if a.SizeX() < 0 || a.SizeY() < 0 {
					t.Fatalf("area %d has negative extent %gx%g", a.ID, a.SizeX(), a.SizeY())
				}
				for _, f := range []float32{a.NorthWest.X, a.NorthWest.Y, a.NorthWest.Z, a.SouthEast.X, a.SouthEast.Y, a.SouthEast.Z, a.NorthEastZ, a.SouthWestZ} {
					if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
						t.Fatalf("area %d has a corner that is not a number", a.ID)
					}
				}

				c := a.Center()
				if !a.Contains2D(c.X, c.Y) {
					t.Fatalf("area %d does not contain its own centre", a.ID)
				}
				if got := a.ClosestPoint(c); got != c {
					t.Fatalf("area %d: closest point to its centre is %v, want %v", a.ID, got, c)
				}
			}
		})
	}
}

// TestZAtCorners pins the height interpolation to the four heights the file
// gives, which is the only part of the geometry that is arithmetic rather than
// a field.
func TestZAtCorners(t *testing.T) {
	a := &Area{
		NorthWest:  Vec3{0, 0, 10},
		SouthEast:  Vec3{100, 200, 40},
		NorthEastZ: 20,
		SouthWestZ: 30,
	}

	cases := []struct {
		name string
		x, y float32
		want float32
	}{
		{"north west corner", 0, 0, 10},
		{"north east corner", 100, 0, 20},
		{"south west corner", 0, 200, 30},
		{"south east corner", 100, 200, 40},
		{"middle", 50, 100, 25},
		{"clamped past the south east corner", 500, 900, 40},
		{"clamped before the north west corner", -500, -900, 10},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.ZAt(c.x, c.y); got != c.want {
				t.Fatalf("ZAt(%g, %g) = %g, want %g", c.x, c.y, got, c.want)
			}
		})
	}
}

// TestParseRefusals covers the negative space: a file this package must not read
// as a mesh.
func TestParseRefusals(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrNotNav},
		{"wrong magic", header(0xDEADBEEF, 16, 2), ErrNotNav},
		{"version 15", header(navMagic, 15, 2), ErrUnsupportedVersion},
		{"version 16 subversion 1", header(navMagic, 16, 1), ErrUnsupportedVersion},
		{"version 17", header(navMagic, 17, 2), ErrUnsupportedVersion},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Parse(c.data); !errors.Is(err, c.want) {
				t.Fatalf("Parse returned %v, want %v", err, c.want)
			}
		})
	}
}

// TestParseTruncated makes sure a short file is an error and never a smaller
// mesh. A misread that runs off the end has to say so; a misread that stays in
// bounds is caught by the end-of-file check instead.
func TestParseTruncated(t *testing.T) {
	full := header(navMagic, SupportedVersion, SupportedSubVersion)

	for cut := 1; cut < len(full); cut++ {
		if _, err := Parse(full[:cut]); err == nil {
			t.Fatalf("a %d byte file parsed", cut)
		}
	}

	if _, err := Parse(append(full, 0)); err == nil {
		t.Fatal("a file with a trailing byte parsed")
	}
}

// header builds the fixed part of a nav file, which is enough for the version
// checks and for the truncation cases.
func header(magic, version, subVersion uint32) []byte {
	b := make([]byte, 0, 32)
	b = binary.LittleEndian.AppendUint32(b, magic)
	b = binary.LittleEndian.AppendUint32(b, version)
	b = binary.LittleEndian.AppendUint32(b, subVersion)
	b = binary.LittleEndian.AppendUint32(b, 0) // bsp size
	b = append(b, 1)                           // analyzed
	b = binary.LittleEndian.AppendUint16(b, 0) // no places
	b = append(b, 0)                           // no unnamed areas
	b = binary.LittleEndian.AppendUint32(b, 0) // no areas
	b = binary.LittleEndian.AppendUint32(b, 0) // no ladders
	return b
}
