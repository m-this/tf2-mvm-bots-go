package tables_test

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

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
