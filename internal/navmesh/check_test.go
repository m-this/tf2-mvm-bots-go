package navmesh

import (
	"fmt"
	"slices"
	"testing"
)

// wantFindings is mvm-z83.25: the map configs as a checked table rather than
// twenty-seven files nobody walked.
//
// One line per finding, "severity rule where", and nothing else. The numbers
// behind each are in testdata/report.txt, which is regenerated and read as a
// diff; this is the classification, and it is meant to fail when a spot moves
// or a rule changes its mind about one. A map with no line here is a map every
// rule passes, which is the statement mvm-tz9 needed somebody to be able to
// make about Decoy.
var wantFindings = map[string][]string{
	"mvm_bigrock": {
		"degraded spot-relocated EngineerNest 1",
		"degraded spot-relocated EngineerNest 2",
		"degraded spot-relocated TeleporterExit 1",
	},
	"mvm_coaltown": {
		"degraded spot-relocated DispenserSpot 2",
	},
	"mvm_mannhattan": {
		"broken spot-in-hole EngineerNest 1",
		"broken spot-in-hole DispenserSpot 2",
		"broken spot-in-hole DispenserSpot 3",
		"note spot-beside-fall EngineerNest 2",
		"note exit-ring-beside-fall EngineerNest 1",
		"note exit-ring-beside-fall EngineerNest 2",
	},
	"mvm_mannworks": {
		"broken spot-in-hole EngineerNest 4",
	},
	"mvm_rottenburg": {
		"broken spot-off-mesh SniperSpot 4",
	},
}

// quietRules is the other half of the table. A rule that has stopped firing
// anywhere is either a bug that was fixed or a check that broke, and the two
// look identical from a passing test, so each one says which it is here.
var quietRules = map[Rule]string{
	RuleNoSpots:     "every config declares sniper spots since mvm-tz9 was fixed on Decoy",
	RuleTooFewExits: "every composition runs one engineer, and the maps that declare an exit declare one",
}

func TestShippedConfigsCheck(t *testing.T) {
	cfgs := loadConfigs(t)

	fired := make(map[Rule]bool)

	for _, c := range cfgs {
		t.Run(c.Map, func(t *testing.T) {
			var m *Mesh
			if haveNav(c.Map) {
				m = loadMap(t, c.Map)
			}

			var got []string
			for _, f := range CheckConfig(m, c) {
				fired[f.Rule] = true
				got = append(got, fmt.Sprintf("%s %s %s %s", f.Rule.Severity(), f.Rule, f.Spot.Kind, f.Spot.Index))
			}

			want := wantFindings[c.Map]
			if !slices.Equal(got, want) {
				t.Errorf("findings changed\n got %q\nwant %q", got, want)
			}
		})
	}

	for _, r := range Rules {
		if fired[r] {
			continue
		}
		if _, ok := quietRules[r]; !ok {
			t.Errorf("%s fires on no shipped config and is not recorded as quiet", r)
		}
	}
	for r := range quietRules {
		if fired[r] {
			t.Errorf("%s is recorded as quiet but fired", r)
		}
	}
}

// TestCheckWithoutMeshStillCounts is what the twenty configs with no nav file
// get. Nothing geometric can be said about them, but the two rules that count
// entries do not need geometry, and both of the bugs they exist for were
// counting bugs.
func TestCheckWithoutMeshStillCounts(t *testing.T) {
	for _, c := range loadConfigs(t) {
		t.Run(c.Map, func(t *testing.T) {
			for _, f := range CheckConfig(nil, c) {
				if f.Rule.NeedsMesh() {
					t.Errorf("%s was reported with no mesh in hand", f.Rule)
				}
			}
		})
	}
}

// TestFindingsSortByRule keeps a report line stable. The findings for one map
// come out grouped, so a diff of the golden is a change in the finding and not
// a change in the order the spots happen to be written in.
func TestFindingsSortByRule(t *testing.T) {
	m := loadMap(t, "mvm_mannhattan")

	var mannhattan *MapConfig
	for _, c := range loadConfigs(t) {
		if c.Map == "mvm_mannhattan" {
			mannhattan = c
		}
	}
	if mannhattan == nil {
		t.Fatal("no mannhattan config")
	}

	found := CheckConfig(m, mannhattan)
	if !slices.IsSortedFunc(found, func(a, b Finding) int { return int(a.Rule) - int(b.Rule) }) {
		t.Errorf("findings are not grouped by rule: %v", found)
	}
}
