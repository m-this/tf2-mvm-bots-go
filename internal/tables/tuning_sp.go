package tables

import (
	"fmt"
	"strings"
)

// SourcePawnTuning is weapon_tuning.sp: the constants the rest of the mod reads
// and the switch that answers for one weapon.
func SourcePawnTuning() []byte {
	var b strings.Builder

	b.WriteString(spHeader("internal/tables/tuning.go"))
	b.WriteString(`
//No opinion: the caller keeps the range its weapon ID produced.
#define RANGE_TUNING_NONE 0.0

#define SOLDIER_ROCKET_SETTLE	750.0
#define DEMO_PIPE_SETTLE		600.0
#define DEMO_PIPE_FIRE_ANYWAY	1400.0

stock float DemoPipeMaxRange()
{
	return DEMO_PIPE_FIRE_ANYWAY;
}

/* Ranges for one weapon. False when the table says nothing about it, and neither output is
touched, so a caller can pass values it already computed */
stock bool GetTunedWeaponRanges(int weapon, float &desired, float &maxRange)
{
	if (!IsValidEntity(weapon) || !HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
		return false;

	switch (GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
	{
`)

	for i, t := range Tunings {
		if t.Section != "" {
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "\t\t//--- %s\n", t.Section)
		}
		if t.Lead != "" {
			b.WriteString(spComment(t.Lead, "\t\t"))
		}
		fmt.Fprintf(&b, "\t\tcase %d: //%s\n\t\t{\n", t.ItemDef, t.Weapon)
		switch {
		case t.Note != "" && t.Block:
			b.WriteString(spComment(t.Note, "\t\t\t"))
		case t.Note != "":
			fmt.Fprintf(&b, "\t\t\t//%s\n", t.Note)
		}
		fmt.Fprintf(&b, "\t\t\tdesired = %s;\n\t\t\tmaxRange = %s;\n\t\t}\n", t.Desired, t.Max)
	}

	b.WriteString(`		default:
		{
			return false;
		}
	}

	return true;
}
`)

	return []byte(b.String())
}
