package tables_test

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// define matches a plugin #define with a value. The value is taken to the end
// of the line and parsed below, so a form this file does not understand fails
// there with the text rather than being missed here.
var define = regexp.MustCompile(`(?m)^#define\s+([A-Z][A-Z0-9_]*)\s+(.+?)\s*$`)

/*
	constantValues reads every #define the table names out of the pinned plugin

Two value forms are understood: a number, and a number added to another define,
which is how TELEPORTER_EXIT_RADIUS_SAFE is written and is the one relation the
plugin already states in code. Anything else fails rather than being skipped,
because a form nobody parsed is a relation nobody checked.
*/
func constantValues(t *testing.T) map[string]float64 {
	t.Helper()

	files := map[string]bool{}
	for _, c := range tables.Constants {
		files[c.File] = true
	}
	raw := map[string]string{}
	for file := range files {
		src := readUpstream(t, "source", "redbots3", file)
		for _, m := range define.FindAllStringSubmatch(src, -1) {
			raw[m[1]] = strings.TrimSpace(m[2])
		}
	}

	values := map[string]float64{}
	var resolve func(name string, depth int) float64
	resolve = func(name string, depth int) float64 {
		if v, ok := values[name]; ok {
			return v
		}
		if depth > len(raw) {
			t.Fatalf("%s resolves in a cycle", name)
		}
		text, ok := raw[name]
		if !ok {
			t.Fatalf("%s is in the table and not in the pinned plugin", name)
		}
		v, err := evaluate(text, func(dep string) float64 { return resolve(dep, depth+1) })
		if err != nil {
			t.Fatalf("%s is defined as %q: %v", name, text, err)
		}
		values[name] = v
		return v
	}
	for _, c := range tables.Constants {
		resolve(c.Name, 0)
	}
	return values
}

// evaluate reads a number, or a name plus a number in parentheses. It is
// deliberately this small: the moment a constant needs more than this, the
// relation it is in should be read again rather than the parser widened.
func evaluate(text string, lookup func(string) float64) (float64, error) {
	text = strings.TrimSpace(strings.Trim(strings.TrimSpace(text), "()"))
	if v, err := strconv.ParseFloat(text, 64); err == nil {
		return v, nil
	}
	name, rest, ok := strings.Cut(text, "+")
	if !ok {
		return 0, fmt.Errorf("not a number and not a name plus a number")
	}
	offset, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
	if err != nil {
		return 0, fmt.Errorf("the added part %q is not a number", strings.TrimSpace(rest))
	}
	return lookup(strings.TrimSpace(name)) + offset, nil
}

// TestEveryRelationNamesConstantsInTheTable is the check on the table rather
// than on the plugin: a relation reading a constant nobody declared would
// compare against whatever the lookup gave it.
func TestEveryRelationNamesConstantsInTheTable(t *testing.T) {
	t.Parallel()

	declared := make([]string, 0, len(tables.Constants))
	for _, c := range tables.Constants {
		declared = append(declared, c.Name)
	}
	for _, r := range tables.Relations {
		for _, name := range r.Uses {
			if !slices.Contains(declared, name) {
				t.Errorf("%q reads %s, which is not in Constants", r.Name, name)
			}
		}
	}
	for _, c := range tables.Constants {
		used := slices.ContainsFunc(tables.Relations, func(r tables.Relation) bool {
			return slices.Contains(r.Uses, c.Name)
		})
		if !used {
			t.Errorf("%s is declared and no relation reads it: the plugin's own comment is a better home for it", c.Name)
		}
	}
}

/*
	TestRelationsHoldInThePinnedPlugin

The proof the bead asks for. Every relation is evaluated against the values the
plugin actually has, so lowering BUSTER_BLAST_RANGE or raising
TELEPORTER_EXIT_RADIUS fails here rather than in a wave weeks later.
*/
func TestRelationsHoldInThePinnedPlugin(t *testing.T) {
	values := constantValues(t)
	lookup := func(name string) float64 {
		v, ok := values[name]
		if !ok {
			t.Fatalf("a relation reads %s, which was never resolved", name)
		}
		return v
	}

	for _, r := range tables.Relations {
		t.Run(r.Name, func(t *testing.T) {
			if r.Holds(lookup) {
				return
			}
			var read strings.Builder
			for _, name := range r.Uses {
				fmt.Fprintf(&read, " %s=%g", name, values[name])
			}
			t.Errorf("%s does not hold:%s\nwhy it exists: %s", r.Statement, read.String(), r.Why)
		})
	}
}

/*
	TestRelationProofCatchesABrokenRelation

The proof about the proof. Every relation is run against the plugin's values with
one constant moved, and has to fail. A relation that holds whatever the numbers
are is a comment with a func literal attached.

The value moved is the first one the relation reads, over a wide spread of
factors, so the direction of the inequality and the slack in it do not have to be
known here. Two of these have a lot of slack and that is fine: the tight ring
sits at 150 against a 400 unit blast, so it takes more than doubling to break it.
*/
func TestRelationProofCatchesABrokenRelation(t *testing.T) {
	values := constantValues(t)

	for _, r := range tables.Relations {
		t.Run(r.Name, func(t *testing.T) {
			moved := r.Uses[0]
			for _, factor := range []float64{0.001, 0.1, 0.5, 2, 10, 1000} {
				lookup := func(name string) float64 {
					if name == moved {
						return values[name] * factor
					}
					return values[name]
				}
				if !r.Holds(lookup) {
					return
				}
			}
			t.Errorf("%s holds with %s at half and at double its value, so it constrains nothing", r.Statement, moved)
		})
	}
}
