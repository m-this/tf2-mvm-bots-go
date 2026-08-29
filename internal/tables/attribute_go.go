package tables

import (
	"fmt"
	"strings"
)

// GoAttributes is the id side for the Go the ranking becomes: one named
// constant per attribute, so a switch over them is a switch over names rather
// than over numbers, and the exhaustiveness argument works the way it does in
// internal/actionsel.
func GoAttributes(pkg string) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "// Code generated from internal/tables/attribute.go. DO NOT EDIT.\n\npackage %s\n\n", pkg)

	b.WriteString(`// Attribute is an upgrade attribute the ranking dispatches on, as the id the
// SourcePawn edge resolved the schema name to.
type Attribute int32

// AttributeNone is the answer for a schema name the ranking has no opinion
// about, which is most of them.
const AttributeNone Attribute = 0

const (
`)
	for _, a := range Attributes {
		fmt.Fprintf(&b, "\t%s Attribute = %d // %s\n", a.GoIdent(), a.ID, a.Name)
	}
	b.WriteString(")\n\n")

	b.WriteString("// AttributeNames is the schema name each id came from, for a message a\n// person has to read.\nvar AttributeNames = map[Attribute]string{\n")
	for _, a := range Attributes {
		fmt.Fprintf(&b, "\t%s: %q,\n", a.GoIdent(), a.Name)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}
