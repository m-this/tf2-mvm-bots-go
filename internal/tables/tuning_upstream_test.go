package tables_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// tunedCase matches one arm of the shipped switch: the item definition, and the
// two ranges it sets.
var tunedCase = regexp.MustCompile(`case (\d+): //([^\n]*)\n\t\t\{(?s:.*?)desired = ([^;]+);\s*\n\s*maxRange = ([^;]+);`)

/*
TestTheTuningTableIsTheShippedOne reads weapon_tuning.sp at the pinned revision
and asks the table the same question for every weapon it names.

The ranges are spelled rather than numbered on purpose: DEMO_PIPE_SETTLE and 600
are the same number and not the same fact, and a table that stored the number
would emit the constant in places the shipped file wrote a literal.
*/
func TestTheTuningTableIsTheShippedOne(t *testing.T) {
	t.Parallel()

	src := readUpstream(t, "source", "redbots3", "weapon_tuning.sp")

	shipped := tunedCase.FindAllStringSubmatch(src, -1)
	if len(shipped) != len(tables.Tunings) {
		t.Fatalf("the shipped file tunes %d weapons and the table holds %d", len(shipped), len(tables.Tunings))
	}

	for i, m := range shipped {
		def, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("item definition %q is not a number", m[1])
		}

		got := tables.Tunings[i]

		switch {
		case int(got.ItemDef) != def:
			t.Errorf("entry %d: the shipped file tunes %d and the table tunes %d", i, def, got.ItemDef)
		case got.Desired != m[3]:
			t.Errorf("%s: the shipped file settles at %s and the table at %s", got.Weapon, m[3], got.Desired)
		case got.Max != m[4]:
			t.Errorf("%s: the shipped file stops firing at %s and the table at %s", got.Weapon, m[4], got.Max)
		}
	}
}
