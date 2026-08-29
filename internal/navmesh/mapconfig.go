package navmesh

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SpotKind is one of the named spot lists a map config holds. The plugin reads
// each into its own ArrayList, and the name is the key in the file.
type SpotKind string

// The spot lists the shipped configs use. A key outside this set is kept as it
// was written, because a config naming something new is a fact to report and not
// an error to swallow.
const (
	SniperSpot     SpotKind = "SniperSpot"
	EngineerNest   SpotKind = "EngineerNest"
	TeleporterExit SpotKind = "TeleporterExit"
	DispenserSpot  SpotKind = "DispenserSpot"
	NestNoTank     SpotKind = "NestNoTank"
	NestTankOnly   SpotKind = "NestTankOnly"
)

// IsGround reports whether spots of this kind name a piece of ground rather than
// a place to look from.
//
// It matters because the two are written at different heights. A building spot
// is where a building goes and so is on the floor, within a step of the nav
// mesh. A sniper spot is copied from where a player stood looking, which is a
// player's eye: 45 units up crouched and 68 standing. Every sniper spot in every
// shipped config sits between 38 and 70 units above the mesh, which is that
// offset and not a fault.
func (k SpotKind) IsGround() bool { return k != SniperSpot }

// Spot is one declared position in a map config, with the index it was written
// under so a finding names the same entry a person would edit.
type Spot struct {
	Kind   SpotKind
	Index  string
	Origin Vec3
	Zone   string
}

// String is the spot as a report line identifies it.
func (s Spot) String() string {
	return fmt.Sprintf("%s %s at %g %g %g", s.Kind, s.Index, s.Origin.X, s.Origin.Y, s.Origin.Z)
}

// MapConfig is one file under configs/defenderbots/map.
type MapConfig struct {
	Map         string
	MovingNests bool
	Composition []string
	Spots       []Spot
}

// SpotsOf returns the declared spots of one kind, in file order.
func (c *MapConfig) SpotsOf(kind SpotKind) []Spot {
	var out []Spot
	for _, s := range c.Spots {
		if s.Kind == kind {
			out = append(out, s)
		}
	}
	return out
}

// ErrBadConfig is returned for a config that is not the KeyValues shape the
// plugin reads.
var ErrBadConfig = errors.New("navmesh: malformed map config")

// LoadMapConfigs reads every .cfg in a directory, which is how the check runs
// over all shipped maps at once instead of one at a time. The result is ordered
// by map name so a report diffs.
func LoadMapConfigs(dir string) ([]*MapConfig, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return nil, fmt.Errorf("navmesh: listing %s: %w", dir, err)
	}
	sort.Strings(entries)

	out := make([]*MapConfig, 0, len(entries))
	for _, path := range entries {
		c, err := LoadMapConfig(path)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	return out, nil
}

// LoadMapConfig reads one map config. The map name is the file name, which is
// how the plugin finds it.
func LoadMapConfig(path string) (*MapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("navmesh: reading %s: %w", path, err)
	}

	c, err := ParseMapConfig(string(data))
	if err != nil {
		return nil, fmt.Errorf("navmesh: %s: %w", path, err)
	}
	c.Map = strings.TrimSuffix(filepath.Base(path), ".cfg")

	return c, nil
}

// ParseMapConfig reads the KeyValues text of one map config.
//
// This is the subset of KeyValues the shipped configs use: quoted keys and
// values, braces, and // to end of line. It is not a general KeyValues reader
// and refuses anything it does not recognise, because a config silently read as
// empty is a map with no spots and a check that passes for the wrong reason.
func ParseMapConfig(text string) (*MapConfig, error) {
	toks, err := tokenizeKV(text)
	if err != nil {
		return nil, err
	}

	p := &kvParser{toks: toks}

	root, err := p.section()
	if err != nil {
		return nil, err
	}
	if root.key != "MapConfig" {
		return nil, fmt.Errorf("%w: root section is %q, not MapConfig", ErrBadConfig, root.key)
	}
	if !p.done() {
		return nil, fmt.Errorf("%w: trailing content after the root section", ErrBadConfig)
	}

	c := &MapConfig{}
	for _, child := range root.children {
		switch {
		case child.children == nil && child.key == "MovingNests":
			c.MovingNests = child.value != "0" && child.value != ""
		case child.children == nil && child.key == "Composition":
			for _, part := range strings.Split(child.value, ",") {
				if part = strings.TrimSpace(part); part != "" {
					c.Composition = append(c.Composition, part)
				}
			}
		case child.children == nil:
			return nil, fmt.Errorf("%w: unknown key %q", ErrBadConfig, child.key)
		default:
			spots, err := spotsFrom(SpotKind(child.key), child)
			if err != nil {
				return nil, err
			}
			c.Spots = append(c.Spots, spots...)
		}
	}

	return c, nil
}

func spotsFrom(kind SpotKind, list *kvNode) ([]Spot, error) {
	out := make([]Spot, 0, len(list.children))

	for _, entry := range list.children {
		if entry.children == nil {
			return nil, fmt.Errorf("%w: %s/%s is a value where a spot was expected", ErrBadConfig, kind, entry.key)
		}

		spot := Spot{Kind: kind, Index: entry.key}
		seenOrigin := false

		for _, field := range entry.children {
			if field.children != nil {
				return nil, fmt.Errorf("%w: %s/%s/%s is a section", ErrBadConfig, kind, entry.key, field.key)
			}
			switch field.key {
			case "origin":
				v, err := parseOrigin(field.value)
				if err != nil {
					return nil, fmt.Errorf("%w: %s/%s: %w", ErrBadConfig, kind, entry.key, err)
				}
				spot.Origin = v
				seenOrigin = true
			case "zone":
				spot.Zone = field.value
			default:
				return nil, fmt.Errorf("%w: %s/%s has unknown field %q", ErrBadConfig, kind, entry.key, field.key)
			}
		}

		if !seenOrigin {
			return nil, fmt.Errorf("%w: %s/%s has no origin", ErrBadConfig, kind, entry.key)
		}
		out = append(out, spot)
	}

	return out, nil
}

func parseOrigin(s string) (Vec3, error) {
	parts := strings.Fields(s)
	if len(parts) != 3 {
		return Vec3{}, fmt.Errorf("origin %q is not three numbers", s)
	}

	var v [3]float32
	for i, p := range parts {
		f, err := strconv.ParseFloat(p, 32)
		if err != nil {
			return Vec3{}, fmt.Errorf("origin %q: %w", s, err)
		}
		v[i] = float32(f)
	}

	return Vec3{v[0], v[1], v[2]}, nil
}

// kvNode is either a value (children nil) or a section (children non-nil,
// possibly empty).
type kvNode struct {
	key      string
	value    string
	children []*kvNode
}

type kvToken struct {
	text    string
	isBrace bool
	line    int
}

type kvParser struct {
	toks []kvToken
	at   int
}

func (p *kvParser) done() bool { return p.at >= len(p.toks) }

func (p *kvParser) section() (*kvNode, error) {
	node, err := p.node()
	if err != nil {
		return nil, err
	}
	if node.children == nil {
		return nil, fmt.Errorf("%w: expected a section, got the value %q", ErrBadConfig, node.key)
	}
	return node, nil
}

func (p *kvParser) node() (*kvNode, error) {
	if p.done() {
		return nil, fmt.Errorf("%w: file ended where a key was expected", ErrBadConfig)
	}

	key := p.toks[p.at]
	if key.isBrace {
		return nil, fmt.Errorf("%w: line %d: %q where a key was expected", ErrBadConfig, key.line, key.text)
	}
	p.at++

	if p.done() {
		return nil, fmt.Errorf("%w: line %d: key %q has nothing after it", ErrBadConfig, key.line, key.text)
	}

	next := p.toks[p.at]
	if !next.isBrace {
		p.at++
		return &kvNode{key: key.text, value: next.text}, nil
	}
	if next.text != "{" {
		return nil, fmt.Errorf("%w: line %d: %q after key %q", ErrBadConfig, next.line, next.text, key.text)
	}
	p.at++

	node := &kvNode{key: key.text, children: []*kvNode{}}
	for {
		if p.done() {
			return nil, fmt.Errorf("%w: section %q is never closed", ErrBadConfig, key.text)
		}
		if t := p.toks[p.at]; t.isBrace && t.text == "}" {
			p.at++
			return node, nil
		}
		child, err := p.node()
		if err != nil {
			return nil, err
		}
		node.children = append(node.children, child)
	}
}

func tokenizeKV(text string) ([]kvToken, error) {
	var out []kvToken
	line := 1

	for i := 0; i < len(text); {
		c := text[i]

		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '/' && i+1 < len(text) && text[i+1] == '/':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case c == '{' || c == '}':
			out = append(out, kvToken{text: string(c), isBrace: true, line: line})
			i++
		case c == '"':
			end := strings.IndexByte(text[i+1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("%w: line %d: unterminated string", ErrBadConfig, line)
			}
			out = append(out, kvToken{text: text[i+1 : i+1+end], line: line})
			line += strings.Count(text[i:i+1+end], "\n")
			i += end + 2
		default:
			return nil, fmt.Errorf("%w: line %d: unquoted %q", ErrBadConfig, line, string(c))
		}
	}

	return out, nil
}
