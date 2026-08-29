package tables

import (
	"fmt"
	"strings"
)

// GoFeatureArms is the arm list the test-bed offers, from the same table the
// convars come from. An arm that names a feature the mod does not have stops
// being possible to write.
func GoFeatureArms(pkg string) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "// Code generated from internal/tables/feature.go. DO NOT EDIT.\n\npackage %s\n\n", pkg)

	b.WriteString(`// Arm is one feature the test-bed can switch off for a comparison.
type Arm struct {
	Name    string
	ConVar  string
	Default bool
	About   string
}

// Arms is every switch the mod ships, in enum order.
var Arms = []Arm{
`)

	for _, f := range Features {
		fmt.Fprintf(&b, "\t{Name: %q, ConVar: %q, Default: %t, About: %q},\n",
			f.Name, f.ConVar(), f.On, f.Description)
	}

	b.WriteString("}\n")
	return []byte(b.String())
}
