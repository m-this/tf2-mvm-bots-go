package spgen

import (
	"fmt"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
)

// The SourcePawn names of the three enums the edge exchanges with the plugin.
//
// They are the Go names of the same constants, listed here rather than read
// out of the Go, because these are the plugin's names: the emitted file is
// what a reviewer of the plugin reads. The values are positional, so the
// length checks below are what stops a constant being added on one side only.
var (
	classNames = []string{
		"ClassUnknown", "ClassScout", "ClassSniper", "ClassSoldier",
		"ClassDemoMan", "ClassMedic", "ClassHeavy", "ClassPyro", "ClassSpy",
		"ClassEngineer",
	}
	roundStateNames = []string{
		"RoundInit", "RoundPregame", "RoundStartGame", "RoundPreround",
		"RoundRunning", "RoundTeamWin", "RoundRestart", "RoundStalemate",
		"RoundGameOver", "RoundBonus", "RoundBetweenRounds",
	}
)

// outcomeNames is the Action enum, in the order of the Go enum, taken from the
// edge so that one outcome is named in exactly one place.
func outcomeNames() []string {
	out := make([]string, 0, len(ActionSelOutcomes))
	for _, o := range ActionSelOutcomes {
		out = append(out, o.Const)
	}
	return out
}

// enums emits the three enums, and refuses to emit one that has lost a
// constant the Go declares.
func enums(cfg Config) (string, error) {
	if got, want := len(classNames), len(actionsel.Classes()); got != want {
		return "", fmt.Errorf("spgen: the edge names %d classes, actionsel declares %d", got, want)
	}
	if got, want := len(roundStateNames), len(actionsel.RoundStates()); got != want {
		return "", fmt.Errorf("spgen: the edge names %d round states, actionsel declares %d", got, want)
	}
	var b strings.Builder
	writeEnum(&b, cfg.Prefix, "Class", classNames)
	writeEnum(&b, cfg.Prefix, "Action", outcomeNames())
	writeEnum(&b, cfg.Prefix, "RoundState", roundStateNames)
	return b.String(), nil
}

func writeEnum(b *strings.Builder, prefix, name string, values []string) {
	fmt.Fprintf(b, "enum %s%s\n{\n", prefix, name)
	for i, v := range values {
		comma := ","
		if i == len(values)-1 {
			comma = ""
		}
		fmt.Fprintf(b, "\t%s%s = %d%s\n", prefix, v, i, comma)
	}
	b.WriteString("};\n\n")
}
