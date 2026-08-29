package tables

import (
	"fmt"
	"strings"
)

/*
	SourcePawnAttributes is the name to id lookup the edge calls once

Every id and every string comes out of one loop over one slice, so the array and
the enum cannot disagree about which name has which id. That is the whole reason
the ranking gets an id at all: 94 StrEqual sites over one text key is forty
chances for a typo that reads as an upgrade nobody wants.

The lookup is a linear scan and stays one: the caller runs it once per upgrade
considered, against a chain that ran up to forty comparisons for the same
answer. A map would be a second data structure to keep in step for no measured
win, and mvm-z83 exists because of facts kept in step by hand.
*/
func SourcePawnAttributes() []byte {
	var b strings.Builder

	b.WriteString(spHeader("internal/tables/attribute.go"))
	b.WriteString(`
/* The upgrade attribute names, and the id the ranking switches on

The ranking used to compare the attribute name against a chain of string literals, once per rank
it might return. The name is a text key from the item schema and the ranking is a pure function of
it, so the key becomes a number at the edge and the ranking becomes a switch.

An id is written down rather than counted, and never reused. A recorded run names the attribute it
ranked, so an id that changed meaning would silently re-read old results. */

enum
{
	ATTRIBUTE_NONE = 0,
`)

	for _, a := range Attributes {
		fmt.Fprintf(&b, "\t%s = %d,\n", a.Enum(), a.ID)
	}
	b.WriteString("};\n\n")

	fmt.Fprintf(&b, "#define ATTRIBUTE_COUNT %d\n\n", len(Attributes))

	b.WriteString("static const char g_strAttributeNames[ATTRIBUTE_COUNT][] =\n{\n")
	for _, a := range Attributes {
		fmt.Fprintf(&b, "\t%q,\n", a.Name)
	}
	b.WriteString("};\n\n")

	b.WriteString(`/* The id for a schema attribute name, or ATTRIBUTE_NONE for one the ranking has no opinion about

ATTRIBUTE_NONE is a real answer and not a failure: the schema has hundreds of attributes and this
table holds the forty eight the ranking dispatches on. */
stock int AttributeID(const char[] name)
{
	for (int i = 0; i < ATTRIBUTE_COUNT; i++)
	{
		if (StrEqual(name, g_strAttributeNames[i]))
			return i + 1;
	}
	
	return ATTRIBUTE_NONE;
}
`)
	return []byte(b.String())
}
