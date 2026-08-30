// Package upgrade is what a bot buys at the upgrade station: the ranking table
// and the SourcePawn it becomes.
package upgrade

import (
	"fmt"
	"sort"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/gen/go/attr"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

/*
The ranking, as SourcePawn.

Three functions, one per layer, each a switch on the attribute id rather than a
chain of string comparisons. The order they are asked in belongs to the caller
and is the shipped one.

Everything here is written from the table in table.go, so the scores exist in one
place and upstream_test.go is what says that place agrees with the plugin.
*/

// enumOf is the SourcePawn constant for an attribute id.
func enumOf(a attr.Attribute) string {
	for _, t := range tables.Attributes {
		if attr.Attribute(t.ID) == a {
			return t.Enum()
		}
	}
	return "ATTRIBUTE_NONE"
}

// cases writes one switch arm per rule, in score order so the file reads as a
// ranking rather than as a list.
func cases(b *strings.Builder, indent string, rules []Rule) {
	sorted := append([]Rule{}, rules...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })

	for _, r := range sorted {
		fmt.Fprintf(b, "%scase %s:\n%s{\n", indent, enumOf(r.Attr), indent)
		switch r.When {
		case Always:
			fmt.Fprintf(b, "%s\treturn %d;\n", indent, r.Score)
		case EngineerDisposable:
			fmt.Fprintf(b, "%s\treturn Feature(FEATURE_ENGINEER_DISPOSABLE) ? %d : %d;\n", indent, r.Score, r.Otherwise)
		case WaveBlast:
			fmt.Fprintf(b, "%s\treturn ResistancePriority(WaveHasExplosiveRobots());\n", indent)
		case WaveBullet:
			fmt.Fprintf(b, "%s\treturn ResistancePriority(WaveHasBulletRobots());\n", indent)
		case WaveFire:
			fmt.Fprintf(b, "%s\treturn ResistancePriority(WaveHasFireRobots());\n", indent)
		}
		fmt.Fprintf(b, "%s}\n", indent)
	}
}

// SourcePawnRanking is upgrade_rank.sp: the three layers the ranking is made of.
func SourcePawnRanking() []byte {
	var b strings.Builder

	b.WriteString(spHeader())
	b.WriteString(`
/* An upgrade no table below recognised

The mod's own answer for every upgrade, kept for the ones the tables do not name. It has to stay
random: a constant would tie every unknown upgrade, and a tie is broken by whichever the game
listed first, so a bot would buy the same wrong thing every wave of every mission. */
stock int UnrankedUpgradePriority()
{
	return GetRandomInt(50, 100);
}

/* What a resistance is worth, given whether the wave will actually deal that damage

Below the damage upgrades on purpose. A team that kills the wave faster takes less of everything,
and the guides all put resistances after the weapon is bought. Above the rest of the tail,
because the alternative is what this mod did before, which was never buying one. */
stock int ResistancePriority(bool wanted)
{
	if (!Feature(FEATURE_WAVE_RESISTANCES))
		return 35;
	
	return wanted ? 210 : 25;
}

/* The upgrade that is the reason to carry this exact weapon, by item definition index

Zero when the weapon in that slot has no opinion, which is most of them: this only names the few
where the loadout, not the class, decides what to buy first. */
stock int UpgradeRankLoadout(int itemDef, int attribute)
{
	switch (itemDef)
	{
`)
	defs := make([]int32, 0, len(Loadout))
	for def := range Loadout {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i] < defs[j] })

	for _, def := range defs {
		fmt.Fprintf(&b, "\t\tcase %d:\n\t\t{\n\t\t\tswitch (attribute)\n\t\t\t{\n", def)
		cases(&b, "\t\t\t\t", Loadout[def])
		b.WriteString("\t\t\t}\n\t\t}\n")
	}

	b.WriteString(`	}

	return 0;
}

/* The metal upgrades an engineer whose gun spends metal wants, which do not hang off the gun

Asked before the slot is looked at at all, because the attribute is on the player rather than on
the weapon that spends it. */
stock int UpgradeRankEngineerMetal(int attribute)
{
	switch (attribute)
	{
`)
	cases(&b, "\t\t", EngineerMetal)
	b.WriteString(`	}

	return 0;
}

/* What this class contributes with, which is not always the weapon in its hands */
stock int UpgradeRankClass(TFClassType pclass, int slot, int attribute)
{
	switch (pclass)
	{
`)
	klasses := make([]Klass, 0, len(Class))
	for k := range Class {
		klasses = append(klasses, k)
	}
	sort.Slice(klasses, func(i, j int) bool { return klasses[i] < klasses[j] })

	for _, k := range klasses {
		rules := Class[k]
		fmt.Fprintf(&b, "\t\tcase %s:\n\t\t{\n", spKlass(k))

		if rules.Split {
			b.WriteString(`			if (slot == TF_LOADOUT_SLOT_PRIMARY || slot == TF_LOADOUT_SLOT_SECONDARY)
			{
				switch (attribute)
				{
`)
			cases(&b, "\t\t\t\t\t", rules.Gun)
			fmt.Fprintf(&b, "\t\t\t\t}\n\n\t\t\t\t//Anything else on the gun is worth less than the cheapest thing the nest wants\n\t\t\t\treturn %d;\n\t\t\t}\n\n", rules.GunFallthrough)
		}

		if rules.MeleeOnly {
			b.WriteString("\t\t\tif (slot != TF_LOADOUT_SLOT_MELEE)\n\t\t\t\treturn 0;\n\n")
		}

		b.WriteString("\t\t\tswitch (attribute)\n\t\t\t{\n")
		cases(&b, "\t\t\t\t", rules.Rest)
		b.WriteString("\t\t\t}\n\t\t}\n")
	}

	b.WriteString(`	}

	return 0;
}

/* Damage first, then what keeps it firing. What a bot buys when nothing above had an opinion */
stock int UpgradeRankGeneral(int attribute)
{
	switch (attribute)
	{
`)
	cases(&b, "\t\t", General)
	b.WriteString(`	}

	return UnrankedUpgradePriority();
}
`)

	return []byte(b.String())
}

// spKlass is the game's own constant for a class.
func spKlass(k Klass) string {
	switch k {
	case KlassScout:
		return "TFClass_Scout"
	case KlassSniper:
		return "TFClass_Sniper"
	case KlassSoldier:
		return "TFClass_Soldier"
	case KlassDemoMan:
		return "TFClass_DemoMan"
	case KlassMedic:
		return "TFClass_Medic"
	case KlassHeavy:
		return "TFClass_Heavy"
	case KlassPyro:
		return "TFClass_Pyro"
	case KlassSpy:
		return "TFClass_Spy"
	case KlassEngineer:
		return "TFClass_Engineer"
	}
	return "TFClass_Unknown"
}

func spHeader() string {
	return "/* Generated from internal/upgrade/table.go. Do not edit.\n\n" +
		"The table it comes from is the only place these scores are written, and its\n" +
		"upstream test is what says they are the shipped ones. */\n"
}
