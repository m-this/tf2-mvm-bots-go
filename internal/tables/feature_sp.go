package tables

import (
	"fmt"
	"strings"
)

// SourcePawnFeatures is features.sp: the enum, the name array, MakeFeature and
// LoadFeatures. The enum and the names come out of one loop over one slice, so
// there is no order for them to disagree about.
func SourcePawnFeatures() []byte {
	var b strings.Builder

	b.WriteString(spHeader("internal/tables/feature.go"))
	b.WriteString(`
/* Ways of playing that can be switched off, so two of them can be compared

Every behaviour in this mod is an argument until somebody measures it, and measuring one means
running the same mission twice with one thing different. That was being done by building two
copies of the mod and keeping them in two directories, which is slow, easy to get wrong, and
impossible to tell apart afterwards from the results alone.

A feature is a named switch with a default. It becomes a convar, so a mission can turn it off in
one line of a config, and the set that was on is written into the wave results, so a file of
numbers says which mod produced it without anybody having to remember.

Adding one is an entry in the Go table and a call to Feature() where the behaviour lives. Removing
one is deleting both, which is the point: a switch nobody has turned off in a month is a
behaviour, and it should stop being a switch. */

enum
{
`)

	for i, f := range Features {
		if i == 0 {
			fmt.Fprintf(&b, "\t%s = 0,\n", f.Enum())
			continue
		}
		fmt.Fprintf(&b, "\t%s,\n", f.Enum())
	}

	b.WriteString("\tFEATURE_COUNT\n}\n")
	b.WriteString(`
/* Same order as the enum above, and both are written by the generator

A name inserted in the wrong place used to rename three convars: "ammo_failover" sat at
FEATURE_WATCH_IDLE_BOTS for a release, which made sm_redbots_feature_watch_lurking_snipers drive
the idle watchdog. An A/B armed the wrong feature and read as a measurement. The constant is now
the name in capitals, so the two cannot part company. */
static const char FEATURE_NAME[FEATURE_COUNT][] =
{
`)

	for i, f := range Features {
		comma := ","
		if i == len(Features)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "\t%q%s\n", f.Name, comma)
	}

	b.WriteString("};\n")
	b.WriteString(`
static ConVar g_arrFeatureConVars[FEATURE_COUNT];
static ConVar g_cvFeaturesActive;

static ConVar MakeFeature(int id, const char[] description, bool on = true)
{
	char name[64]; Format(name, sizeof(name), "sm_redbots_feature_%s", FEATURE_NAME[id]);

	/* A feature ships on once it has been measured, and off until then

	The switch exists to turn something off and measure the difference, and that is the whole point
	of it: a behaviour that has not cleared the spread of the arm it was measured against is not
	yet a behaviour this mod claims. See the rule in docs/testbed-metrics.md. */
	return CreateConVar(name, on ? "1" : "0", description, FCVAR_NOTIFY);
}

void LoadFeatures()
{
`)

	for i, f := range Features {
		if i > 0 {
			b.WriteString("\n")
		}
		if f.Note != "" {
			b.WriteString(spComment(f.Note, "\t"))
		}
		fmt.Fprintf(&b, "\tg_arrFeatureConVars[%s] = MakeFeature(%s,\n", f.Enum(), f.Enum())
		if f.On {
			fmt.Fprintf(&b, "\t\t%q);\n", f.Description)
			continue
		}
		fmt.Fprintf(&b, "\t\t%q, false);\n", f.Description)
	}

	fmt.Fprintf(&b, `
	/* What is on, as one string, for whoever reads the results later

	Written rather than read: nothing in the mod uses it. It exists so the statistics plugin can
	put it in the file, because a run whose settings are not recorded is a run that cannot be
	compared with anything. */
	g_cvFeaturesActive = CreateConVar(%q, "",
		"The features that are on, comma separated. Set by the mod, not by you.", FCVAR_NONE);
}

//A feature nobody switched is on, so a config that names none of them gets the mod as shipped
bool Feature(int id)
{
	if (id < 0 || id >= FEATURE_COUNT || g_arrFeatureConVars[id] == null)
		return true;

	return g_arrFeatureConVars[id].BoolValue;
}

/* Publish the set that is on

Called when a wave begins rather than once at map start: a config file executes at its own pace
and a late loaded plugin misses it entirely, so the answer earlier than this is the defaults
rather than what the server was asked for. */
void PublishActiveFeatures()
{
	if (g_cvFeaturesActive == null)
		return;

	char list[512];

	for (int i = 0; i < FEATURE_COUNT; i++)
	{
		if (!Feature(i))
			continue;

		if (list[0] != '\0')
			StrCat(list, sizeof(list), ",");

		StrCat(list, sizeof(list), FEATURE_NAME[i]);
	}

	/* The three BLU scales ride along, because they are the other thing that can
	make one run differ from another and a results file has to say so. They are
	convars rather than switches: 1.0 is off, so a scale that is set at all is
	worth recording, and there is nothing to record when none is. */
	BluAssist_Describe(list, sizeof(list));

	g_cvFeaturesActive.SetString(list);
}
`, FeaturesActiveConVar)

	return []byte(b.String())
}

func spHeader(from string) string {
	return "/* Generated from " + from + ". Do not edit.\n\n" +
		"The table it comes from is the only place these names are written. */\n"
}

// spComment wraps a note in the block form the plugin uses, indented to match
// the statement it sits above.
func spComment(note, indent string) string {
	lines := strings.Split(note, "\n")

	var b strings.Builder
	for i, line := range lines {
		switch {
		case i == 0:
			b.WriteString(indent + "/* " + line + "\n")
		case line == "":
			b.WriteString("\n")
		default:
			b.WriteString(indent + line + "\n")
		}
	}

	trimmed := strings.TrimSuffix(b.String(), "\n")
	return trimmed + " */\n"
}
