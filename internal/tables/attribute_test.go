package tables_test

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

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
