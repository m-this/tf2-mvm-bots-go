/*
Package generated is the whole of what the generators write.

cmd/gen writes it to gen/, and internal/adopt places the part the plugin tree
commits. One list, so a generator that gains an output cannot leave it
unwatched: a file is shipped, or it is proof, and Files is where that is
decided.
*/
package generated

import (
	"fmt"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
	"github.com/m-this/tf2-mvm-bots-go/internal/upgrade"
)

// Files is every generated file, keyed by its path under the output directory.
// A generator that emits a file not listed here does not exist as far as the
// reproducibility check is concerned.
func Files(root string) (map[string][]byte, error) {
	sel, err := spgen.EmitActionSel()
	if err != nil {
		return nil, fmt.Errorf("emitting action selection: %w", err)
	}
	bodies, err := body.Generate(root)
	if err != nil {
		return nil, fmt.Errorf("emitting bodies: %w", err)
	}
	out := map[string][]byte{
		"sourcepawn/actionsel.sp":          []byte(sel.Data),
		"sourcepawn/actionsel_dispatch.sp": []byte(sel.Dispatch),
		"sourcepawn/attributes.sp":         tables.SourcePawnAttributes(),
		"sourcepawn/features.sp":           tables.SourcePawnFeatures(),
		"sourcepawn/threat_priority.sp":    spgen.EmitThreatPriority(),
		"sourcepawn/upgrade_rank.sp":       upgrade.SourcePawnRanking(),
		"sourcepawn/weapon_tuning.sp":      tables.SourcePawnTuning(),
		"sourcepawn/wave_write.sp":         tables.SourcePawnWaveWriter(),
		"go/arms/arms.go":                  tables.GoFeatureArms("arms"),
		"go/attr/attr.go":                  tables.GoAttributes("attr"),
		"go/wave/wave.go":                  tables.GoWaveParser("wave"),
		"go/injectors/injectors.go":        tables.GoInjectors("injectors"),
	}
	for name, source := range bodies {
		if _, taken := out[name]; taken {
			return nil, fmt.Errorf("two generators write %s", name)
		}
		out[name] = source
	}
	return out, nil
}

/*
Proof is the SourcePawn that is generated and compiled and ships nowhere.

The action selection table predates the body generator and GetDesiredBotAction
is a body now, in internal/body/dispatch; the table stays as the exhaustive
proof over every reachable combination. The bodies flagged Proof in
internal/body are the differential harness's subjects. Nothing else in the
output may be missing from the plugin tree.
*/
func Proof(name string) bool {
	switch name {
	case "sourcepawn/actionsel.sp", "sourcepawn/actionsel_dispatch.sp":
		return true
	}
	for _, b := range body.All {
		if b.Proof && (name == b.Out || name == b.Hooks) {
			return true
		}
	}
	return false
}
