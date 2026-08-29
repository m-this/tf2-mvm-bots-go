package tables_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// upstreamFeature is one feature as features.sp has it today, read back out of
// the three places the file writes it.
type upstreamFeature struct {
	enum        string
	name        string
	convar      string
	description string
	on          bool
}

var (
	reEnumBlock = regexp.MustCompile(`(?s)\nenum\n\{\n(.*?)\n\}`)
	reEnumEntry = regexp.MustCompile(`^\s*(FEATURE_[A-Z0-9_]+)(?:\s*=\s*\d+)?,?\s*$`)
	reNameBlock = regexp.MustCompile(`(?s)FEATURE_NAME\[FEATURE_COUNT\]\[\]\s*=\s*\{(.*?)\};`)
	reNameEntry = regexp.MustCompile(`"([^"]*)"`)
	reMakeFeat  = regexp.MustCompile(`g_arrFeatureConVars\[(FEATURE_[A-Z0-9_]+)\]\s*=\s*MakeFeature\((FEATURE_[A-Z0-9_]+),\s*\n\s*"((?:[^"\\]|\\.)*)"(\s*,\s*false)?\);`)
	rePrefix    = regexp.MustCompile(`Format\(name, sizeof\(name\), "([^"]*)%s"`)
)

// parseFeaturesSP reads features.sp the way the compiler does not: it checks
// that the enum, the name array and the LoadFeatures calls agree, and returns
// what they say.
func parseFeaturesSP(t *testing.T, src string) []upstreamFeature {
	t.Helper()

	enumBlock := reEnumBlock.FindStringSubmatch(src)
	if enumBlock == nil {
		t.Fatal("no enum block in features.sp")
	}

	var enums []string
	for _, line := range strings.Split(enumBlock[1], "\n") {
		m := reEnumEntry.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("unparsed enum line %q", line)
		}
		if m[1] == "FEATURE_COUNT" {
			continue
		}
		enums = append(enums, m[1])
	}

	nameBlock := reNameBlock.FindStringSubmatch(src)
	if nameBlock == nil {
		t.Fatal("no FEATURE_NAME array in features.sp")
	}

	names := make([]string, 0, len(tables.Features))
	for _, m := range reNameEntry.FindAllStringSubmatch(nameBlock[1], -1) {
		names = append(names, m[1])
	}

	if len(enums) != len(names) {
		t.Fatalf("features.sp has %d enum entries and %d names", len(enums), len(names))
	}

	prefix := rePrefix.FindStringSubmatch(src)
	if prefix == nil {
		t.Fatal("no convar name format in MakeFeature")
	}

	calls := map[string]upstreamFeature{}
	for _, m := range reMakeFeat.FindAllStringSubmatch(src, -1) {
		if m[1] != m[2] {
			t.Fatalf("MakeFeature stores %s under %s", m[2], m[1])
		}
		description, err := strconv.Unquote(`"` + m[3] + `"`)
		if err != nil {
			t.Fatalf("description of %s: %v", m[1], err)
		}
		calls[m[1]] = upstreamFeature{description: description, on: m[4] == ""}
	}

	out := make([]upstreamFeature, 0, len(enums))
	for i, enum := range enums {
		call, ok := calls[enum]
		if !ok {
			t.Fatalf("%s is in the enum but LoadFeatures never makes it", enum)
		}
		call.enum = enum
		call.name = names[i]
		call.convar = prefix[1] + names[i]
		out = append(out, call)
	}

	if len(out) != len(calls) {
		t.Fatalf("LoadFeatures makes %d features for %d enum entries", len(calls), len(out))
	}
	return out
}

// TestFeatureTableRoundTrips is the proof the bead asks for: every feature the
// plugin ships today comes back out of the Go table with the same convar name,
// the same default and the same description, byte for byte.
func TestFeatureTableRoundTrips(t *testing.T) {
	t.Parallel()

	upstream := parseFeaturesSP(t, readUpstream(t, "source", "redbots3", "features.sp"))

	if len(upstream) != len(tables.Features) {
		t.Fatalf("features.sp has %d features, the table has %d", len(upstream), len(tables.Features))
	}
	if len(upstream) != 22 {
		t.Fatalf("features.sp has %d features, the bead says 22", len(upstream))
	}

	for i, want := range upstream {
		got := tables.Features[i]

		t.Run(want.name, func(t *testing.T) {
			if got.Name != want.name {
				t.Errorf("name at index %d: got %q, features.sp has %q", i, got.Name, want.name)
			}
			if got.Enum() != want.enum {
				t.Errorf("enum: got %q, features.sp has %q", got.Enum(), want.enum)
			}
			if got.ConVar() != want.convar {
				t.Errorf("convar: got %q, features.sp has %q", got.ConVar(), want.convar)
			}
			if got.On != want.on {
				t.Errorf("default: got on=%t, features.sp has on=%t", got.On, want.on)
			}
			if got.Description != want.description {
				t.Errorf("description:\n got %q\nwant %q", got.Description, want.description)
			}
		})
	}
}

// TestGeneratedFeaturesSPRoundTrips runs the generated file back through the
// same parser. The enum, the names and the convars have to survive generation,
// not only the table.
func TestGeneratedFeaturesSPRoundTrips(t *testing.T) {
	t.Parallel()

	upstream := parseFeaturesSP(t, readUpstream(t, "source", "redbots3", "features.sp"))
	generated := parseFeaturesSP(t, string(tables.SourcePawnFeatures()))

	if len(generated) != len(upstream) {
		t.Fatalf("generated %d features, features.sp has %d", len(generated), len(upstream))
	}

	for i := range upstream {
		if generated[i] != upstream[i] {
			t.Errorf("feature %d:\n got %+v\nwant %+v", i, generated[i], upstream[i])
		}
	}
}

// TestFeatureNamesAreUnique. Two features sharing a name share a convar, and
// the one loaded second wins silently.
func TestFeatureNamesAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	for i, f := range tables.Features {
		if first, ok := seen[f.Name]; ok {
			t.Errorf("feature %q at index %d and %d", f.Name, first, i)
		}
		seen[f.Name] = i
	}
}

// TestFeatureNamesAreConVarSafe. A capital or a space in a name makes a convar
// nobody can set from a config.
func TestFeatureNamesAreConVarSafe(t *testing.T) {
	t.Parallel()

	safe := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, f := range tables.Features {
		if !safe.MatchString(f.Name) {
			t.Errorf("feature name %q is not usable in a convar", f.Name)
		}
		if f.Description == "" {
			t.Errorf("feature %q has no description", f.Name)
		}
	}
}
