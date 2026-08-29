package tables_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// strEqualAttribute matches the dispatch the ranking is written with today.
var strEqualAttribute = regexp.MustCompile(`StrEqual\(attribute, "([^"]+)"\)`)

// upgradeAttributeNames is every name behavior/upgrade.sp compares against, in
// the order it first mentions them.
func upgradeAttributeNames(t *testing.T) []string {
	t.Helper()

	src := readUpstream(t, "source", "redbots3", "behavior", "upgrade.sp")
	var names []string
	for _, m := range strEqualAttribute.FindAllStringSubmatch(src, -1) {
		if !slices.Contains(names, m[1]) {
			names = append(names, m[1])
		}
	}
	if len(names) == 0 {
		t.Fatal("no StrEqual(attribute, ...) sites in the pinned upgrade.sp: the dispatch changed shape")
	}
	return names
}

/*
	TestAttributeTableCoversTheRanking

The table has to hold exactly the names the ranking dispatches on. A name in the
plugin and not here is a rank the Go switch would never reach; a name here and
not in the plugin is an id standing for nothing, which is the kind of leftover
that gets read as meaningful later.
*/
func TestAttributeTableCoversTheRanking(t *testing.T) {
	shipped := upgradeAttributeNames(t)

	declared := make([]string, 0, len(tables.Attributes))
	for _, a := range tables.Attributes {
		declared = append(declared, a.Name)
	}

	for _, name := range shipped {
		if !slices.Contains(declared, name) {
			t.Errorf("upgrade.sp ranks %q and the table has no id for it", name)
		}
	}
	for _, name := range declared {
		if !slices.Contains(shipped, name) {
			t.Errorf("the table has %q and the pinned upgrade.sp does not rank it", name)
		}
	}
	t.Logf("%d attribute names, %d StrEqual sites in the shipped ranking", len(declared), len(shipped))
}

// TestAttributeIDsAreUniqueAndPositive covers the negative space of the id
// rule. Zero is ATTRIBUTE_NONE and belongs to no attribute, and a repeated id
// would make two names the same fact.
func TestAttributeIDsAreUniqueAndPositive(t *testing.T) {
	t.Parallel()

	seen := map[int32]string{}
	idents := map[string]string{}
	for _, a := range tables.Attributes {
		if a.ID <= 0 {
			t.Errorf("%q has id %d, and zero is ATTRIBUTE_NONE", a.Name, a.ID)
		}
		if other, dup := seen[a.ID]; dup {
			t.Errorf("id %d is both %q and %q", a.ID, other, a.Name)
		}
		seen[a.ID] = a.Name

		for _, ident := range []string{a.Enum(), a.GoIdent()} {
			if other, dup := idents[ident]; dup {
				t.Errorf("%q and %q both spell %s", other, a.Name, ident)
			}
			idents[ident] = a.Name
		}
	}
}

/*
	TestAttributeIDsSurviveAnInsert

The rule the ids exist for. Inserting a name in the middle of the slice must
leave every other id where it was, because a run records the attribute it ranked
and an id that changed meaning re-reads old results as something else.
*/
func TestAttributeIDsSurviveAnInsert(t *testing.T) {
	t.Parallel()

	before := map[string]int32{}
	for _, a := range tables.Attributes {
		before[a.Name] = a.ID
	}

	inserted := slices.Clone(tables.Attributes)
	inserted = slices.Insert(inserted, len(inserted)/2, tables.Attribute{ID: 9999, Name: "a name nobody ranks"})

	for _, a := range inserted {
		if was, existed := before[a.Name]; existed && was != a.ID {
			t.Errorf("%q was id %d and is %d after an insert above it", a.Name, was, a.ID)
		}
	}
}
