package tables_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// spFeature is one feature as features.sp has it, read back out of the three
// places the file writes it.
type spFeature struct {
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
func parseFeaturesSP(t *testing.T, src string) []spFeature {
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

	calls := map[string]spFeature{}
	for _, m := range reMakeFeat.FindAllStringSubmatch(src, -1) {
		if m[1] != m[2] {
			t.Fatalf("MakeFeature stores %s under %s", m[2], m[1])
		}
		description, err := strconv.Unquote(`"` + m[3] + `"`)
		if err != nil {
			t.Fatalf("description of %s: %v", m[1], err)
		}
		calls[m[1]] = spFeature{description: description, on: m[4] == ""}
	}

	out := make([]spFeature, 0, len(enums))
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

/*
	TestGeneratedFeaturesSPRoundTrips

The generated file read back by a parser that knows nothing about the generator:
the enum, the name array and the LoadFeatures calls have to agree with each
other and say what the table says.

The three of them are written from one table and are three places a feature can
go missing, which is the bug this started as: a name inserted in the middle of
the enum renames every convar below it and nothing complains.
*/
func TestGeneratedFeaturesSPRoundTrips(t *testing.T) {
	t.Parallel()

	generated := parseFeaturesSP(t, string(tables.SourcePawnFeatures()))

	if len(generated) != len(tables.Features) {
		t.Fatalf("the generated file holds %d features and the table has %d", len(generated), len(tables.Features))
	}

	for i, f := range tables.Features {
		want := spFeature{
			enum:        f.Enum(),
			name:        f.Name,
			convar:      f.ConVar(),
			description: f.Description,
			on:          f.On,
		}
		if got := generated[i]; got != want {
			t.Errorf("feature %d:\n got %+v\nwant %+v", i, got, want)
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
