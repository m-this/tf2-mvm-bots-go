package upgrade_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
	"github.com/m-this/tf2-mvm-bots-go/internal/upgrade"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
The ranking table against the ranking it replaces.

The shipped file writes the same three layers as ninety-four string comparisons.
This reads them back out of it at the pinned revision and asks the Go table the
same question, for every attribute the shipped file names and for every class,
slot and condition it distinguishes.

It is the whole proof of this package. The generated SourcePawn is a switch and
the shipped one is a comparison chain, so the two cannot be compared as text or
as a sequence of calls: what has to match is the answer, and the answer is what
this compares.
*/

var (
	strEqualScore = regexp.MustCompile(`StrEqual\(attribute, "([^"]+)"\)\)\s*return\s+(-?\d+);`)
	loadoutCase   = regexp.MustCompile(`case (\d+): //`)
	classCase     = regexp.MustCompile(`case TFClass_(\w+):`)
)

// shippedFunc is one static function of the shipped file, brace matched.
func shippedFunc(t *testing.T, src, name string) string {
	t.Helper()

	start := strings.Index(src, "static int "+name+"(")
	if start < 0 {
		t.Fatalf("%s is not in the shipped file any more", name)
	}

	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1]
			}
		}
	}
	t.Fatalf("%s never closes", name)
	return ""
}

// attrOf is the id the tables package gives a schema name.
func attrOf(t *testing.T, name string) tables.Attribute {
	t.Helper()

	for _, a := range tables.Attributes {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("the ranking names %q and internal/tables does not", name)
	return tables.Attribute{}
}

// rulesOf is every (attribute, score) the text holds, in order.
func rulesOf(t *testing.T, text string) []upgrade.Rule {
	t.Helper()

	found := strEqualScore.FindAllStringSubmatch(text, -1)
	out := make([]upgrade.Rule, 0, len(found))

	for _, m := range found {
		score, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("the score for %q is not a number: %v", m[1], err)
		}
		//nolint:gosec // G109: the scores are three digits, written by hand in the shipped file
		out = append(out, upgrade.Rule{Attr: attrOf(t, m[1]), Score: upgrade.Score(score)})
	}
	return out
}

func TestTheLoadoutTableIsTheShippedOne(t *testing.T) {
	t.Parallel()

	src := readUpstream(t)
	body := shippedFunc(t, src, "LoadoutUpgradePriority")

	// Everything before the slot test is the engineer's metal, which hangs
	// off the class rather than off the gun.
	split := strings.Index(body, "if (slot < TF_LOADOUT_SLOT_PRIMARY")
	if split < 0 {
		t.Fatal("the shipped file no longer bounds the loadout switch by slot")
	}
	compare(t, "the engineer's metal", rulesOf(t, body[:split]), upgrade.EngineerMetal)

	cases := loadoutCase.FindAllStringSubmatchIndex(body[split:], -1)
	if len(cases) != len(upgrade.Loadout) {
		t.Fatalf("the shipped file ranks %d weapons and the table holds %d", len(cases), len(upgrade.Loadout))
	}

	for i, c := range cases {
		def, err := strconv.Atoi(body[split:][c[2]:c[3]])
		if err != nil {
			t.Fatalf("item definition %q is not a number", body[split:][c[2]:c[3]])
		}

		end := len(body[split:])
		if i+1 < len(cases) {
			end = cases[i+1][0]
		}

		//nolint:gosec // G115: item definition indexes are four digits in the shipped file
		compare(t, fmt.Sprintf("item definition %d", def),
			rulesOf(t, body[split:][c[0]:end]), upgrade.Loadout[int32(def)])
	}
}

func TestTheGeneralTableIsTheShippedOne(t *testing.T) {
	t.Parallel()

	src := readUpstream(t)
	body := shippedFunc(t, src, "GeneralUpgradePriority")

	// The three resistances are not constants and are checked by
	// TestTheResistancesAreThePricedOnes instead.
	var plain []upgrade.Rule
	for _, r := range upgrade.General {
		if r.When == upgrade.Always {
			plain = append(plain, r)
		}
	}

	compare(t, "the general table", rulesOf(t, body), plain)
}

func TestTheClassTablesAreTheShippedOnes(t *testing.T) {
	t.Parallel()

	src := readUpstream(t)
	body := shippedFunc(t, src, "ClassUpgradePriority")

	cases := classCase.FindAllStringSubmatchIndex(body, -1)
	if len(cases) != len(upgrade.Class) {
		t.Fatalf("the shipped file ranks %d classes and the table holds %d", len(cases), len(upgrade.Class))
	}

	for i, c := range cases {
		name := body[c[2]:c[3]]

		end := len(body)
		if i+1 < len(cases) {
			end = cases[i+1][0]
		}

		text := body[c[0]:end]
		klass := klassOf(t, name)
		rules := upgrade.Class[klass]

		// The disposable sentry is behind a feature switch and is the one
		// rule in these tables that is not a constant.
		var want []upgrade.Rule
		for _, r := range append(append([]upgrade.Rule{}, rules.Gun...), rules.Rest...) {
			if r.When == upgrade.Always {
				want = append(want, r)
			}
		}

		compare(t, name, rulesOf(t, text), want)
	}
}

// klassOf maps the shipped constant's tail onto the Go one.
func klassOf(t *testing.T, name string) upgrade.Klass {
	t.Helper()

	for klass, spelling := range map[upgrade.Klass]string{
		upgrade.KlassScout: "Scout", upgrade.KlassSniper: "Sniper",
		upgrade.KlassSoldier: "Soldier", upgrade.KlassDemoMan: "DemoMan",
		upgrade.KlassMedic: "Medic", upgrade.KlassHeavy: "Heavy",
		upgrade.KlassPyro: "Pyro", upgrade.KlassSpy: "Spy",
		upgrade.KlassEngineer: "Engineer",
	} {
		if spelling == name {
			return klass
		}
	}
	t.Fatalf("the shipped file ranks TFClass_%s and this test does not know it", name)
	return upgrade.KlassUnknown
}

// compare is order-insensitive on purpose: the shipped file writes its rules in
// the order it tests them and the table holds the same facts, so what has to
// match is the set of (attribute, score) pairs.
func compare(t *testing.T, what string, shipped, table []upgrade.Rule) {
	t.Helper()

	got := map[tables.Attribute]upgrade.Score{}
	for _, r := range table {
		got[r.Attr] = r.Score
	}

	for _, r := range shipped {
		score, ranked := got[r.Attr]
		if !ranked {
			t.Errorf("%s: the shipped file ranks %q at %d and the table does not rank it", what, r.Attr.Name, r.Score)
			continue
		}
		if score != r.Score {
			t.Errorf("%s: the shipped file ranks %q at %d and the table at %d", what, r.Attr.Name, r.Score, score)
		}
		delete(got, r.Attr)
	}

	for a, score := range got {
		t.Errorf("%s: the table ranks %q at %d and the shipped file does not rank it", what, a.Name, score)
	}
}

func readUpstream(t *testing.T) string {
	t.Helper()

	upstream.SkipOrFail(t)
	body, err := upstream.Read("source", "redbots3", "behavior", "upgrade.sp")
	if err != nil {
		t.Fatalf("reading upgrade.sp at %s: %v", upstream.Rev, err)
	}
	return body
}
