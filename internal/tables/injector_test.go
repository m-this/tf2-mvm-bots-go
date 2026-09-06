package tables_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// createdConVar is a convar the faults body makes, whatever helper it uses.
var createdConVar = regexp.MustCompile(`engine\.Create\w*ConVar\("(sm_redbots_debug_[a-z_]+)"`)

// faultsBody is the package the injectors live in. The table is read against
// the body's own source rather than against the generated SourcePawn, because
// the body is where somebody adds one.
const faultsBody = "../body/faults/faults.go"

func declaredInTheBody(t *testing.T) []string {
	t.Helper()

	source, err := os.ReadFile(filepath.Clean(faultsBody))
	if err != nil {
		t.Fatalf("reading %s: %v", faultsBody, err)
	}

	found := createdConVar.FindAllStringSubmatch(string(source), -1)

	out := make([]string, 0, len(found))
	for _, m := range found {
		out = append(out, m[1])
	}
	return out
}

/*
	TestEveryInjectorIsDeclaredOnBothSides

The table is what the test-bed knows and the body is what the server creates. A
row with no convar is an arm that sets nothing and reads as a measurement, which
is the fault mvm-zx0 sat on: its injector was never armed and nothing said so.
A convar with no row is a fault the test-bed cannot name.
*/
func TestEveryInjectorIsDeclaredOnBothSides(t *testing.T) {
	t.Parallel()

	body := declaredInTheBody(t)
	if len(body) == 0 {
		t.Fatalf("no debug convars found in %s, so this proves nothing", faultsBody)
	}

	table := make([]string, 0, len(tables.Injectors))
	for _, i := range tables.Injectors {
		table = append(table, i.ConVar())
	}

	for _, convar := range table {
		if !slices.Contains(body, convar) {
			t.Errorf("%s is in the table and the plugin never creates it", convar)
		}
	}
	for _, convar := range body {
		if !slices.Contains(table, convar) {
			t.Errorf("%s is created by the plugin and no table row names it", convar)
		}
	}
	if !slices.Equal(table, body) {
		t.Errorf("the order differs:\n table %v\n body  %v", table, body)
	}
}

// TestOnlyAValueArmsAnInjector. Zero is off, a class name selects rather than
// arms, and anything else is on.
func TestOnlyAValueArmsAnInjector(t *testing.T) {
	t.Parallel()

	wedge, ok := tables.InjectorByConVar("sm_redbots_debug_wedge_seconds")
	if !ok {
		t.Fatal("the wedge injector is not in the table")
	}
	for value, want := range map[string]bool{"": false, "0": false, "0.0": false, "60": true, "0.5": true} {
		if got := wedge.Arms(value); got != want {
			t.Errorf("wedge_seconds=%q armed %v, want %v", value, got, want)
		}
	}

	class, ok := tables.InjectorByConVar("sm_redbots_debug_wedge_class")
	if !ok {
		t.Fatal("the wedge class is not in the table")
	}
	if class.Arms("engineer") {
		t.Error("naming a class armed a fault on its own")
	}

	if _, ok := tables.InjectorByConVar("sm_redbots_debug_nothing"); ok {
		t.Error("an injector the mod does not have was found in the table")
	}
}

// TestEveryInjectorSaysWhichBugItIsFor. An injector with no bead behind it is a
// switch somebody invented, and those get worked around rather than respected.
func TestEveryInjectorSaysWhichBugItIsFor(t *testing.T) {
	t.Parallel()

	for _, i := range tables.Injectors {
		if i.Why == "" || i.About == "" {
			t.Errorf("%s has no reason written down", i.Name)
		}
	}
}
