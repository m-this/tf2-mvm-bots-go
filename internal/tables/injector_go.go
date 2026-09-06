package tables

import (
	"fmt"
	"strings"
)

// GoInjectors is the fault list the test-bed reads, from the same table the
// plugin's convars come from. An arm that names an injector the mod does not
// have stops being possible to write.
func GoInjectors(pkg string) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "// Code generated from internal/tables/injector.go. DO NOT EDIT.\n\npackage %s\n\nimport \"strings\"\n\n", pkg)

	b.WriteString(`// Injector is one fault the test-bed can make the mod produce.
type Injector struct {
	Name   string
	ConVar string
	Kind   string
	About  string
}

// Injectors is every one the mod has, in the order it creates them.
var Injectors = []Injector{
`)

	for _, i := range Injectors {
		fmt.Fprintf(&b, "\t{Name: %q, ConVar: %q, Kind: %q, About: %q},\n", i.Name, i.ConVar(), string(i.Kind), i.About)
	}

	b.WriteString(`}

/*
Armed is every injector the cvars turn on, by name.

A class name selects which bot the others take rather than arming anything, so
it is never one of them, and every other kind is off at zero. What this returns
goes into the run record: a results file whose arm is not written down is one
mvm-81n says nobody can read afterwards.
*/
func Armed(cvars string) []string {
	var out []string

	for pair := range strings.SplitSeq(cvars, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(pair), "=")
		if !found {
			continue
		}

		key, value = strings.TrimSpace(key), strings.TrimSpace(value)

		for _, i := range Injectors {
			if i.ConVar != key || i.Kind == "name" {
				continue
			}

			switch value {
			case "", "0", "0.0", "false":
			default:
				out = append(out, i.Name)
			}
		}
	}

	return out
}
`)

	return []byte(b.String())
}
