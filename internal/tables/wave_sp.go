package tables

import (
	"fmt"
	"strings"
)

// The format string is broken across lines at a width close to the rest of the
// plugin. Nothing depends on where the breaks fall: FormatEx sees one string.
const spLineWidth = 100

// SourcePawnWaveWriter is the FormatEx that writes one wave line, with its
// argument list in the same order as the table.
func SourcePawnWaveWriter() []byte {
	var b strings.Builder

	b.WriteString(spHeader("internal/tables/wave.go"))
	b.WriteString(`
/* One line for the wave, with everything that was counted while it ran
 *
 * The duration is the honest number to compare runs on: a wave that is cleared slowly is a team
 * that nearly lost it, and a change that clears the same waves faster is a change that worked
 *
 * Not static, because spcomp scopes static to the file and this one is included: the helpers it
 * calls live in the plugin and the callers of it do too. */
void WriteWaveResult(const char[] result)
{
	/* A wave nobody played is not a result
	 *
	 * The game ends a wave when the round resets, which it does when the server restarts, so a
	 * restart wrote a row of zeros into the file. run.sh counts rows, so that row was the run: it
	 * stopped twenty seconds in and reported a wave lost that never began. Only a wave with a
	 * beginning is written.
	 */
	if (g_flWaveStart <= 0.0)
	{
		return;
	}

	CollectScoreboardHealing();

	float duration = GetGameTime() - g_flWaveStart;

	char featuresFired[512];
	FeaturesFiredSince(featuresFired, sizeof(featuresFired));

	char line[STATS_LINE_LENGTH];
	FormatEx(line, sizeof(line),
`)

	format := spFormatLines()
	for i, line := range format {
		tail := ""
		if i == len(format)-1 {
			tail = ","
		}
		if i == 0 {
			fmt.Fprintf(&b, "\t\t%s%s\n", line, tail)
			continue
		}
		fmt.Fprintf(&b, "\t\t... %s%s\n", line, tail)
	}

	var args []string
	for _, f := range WaveRecord {
		if f.Literal != "" {
			continue
		}
		args = append(args, f.SP)
	}

	for i, arg := range args {
		tail := ","
		if i == len(args)-1 {
			tail = ");"
		}
		fmt.Fprintf(&b, "\t\t%s%s\n", arg, tail)
	}

	/* The tail is the rest of the shipped function, and it is here rather
	than left to the caller because the caller is a generated file: what
	ships has to be the whole of WriteWaveResult or the plugin loses the
	perf line, the engineer lines and the reset that stops the next round
	reset writing a row of zeros.

	None of it is table driven. The wave record is the FormatEx above; the
	perf line is about the machine rather than the bots, and it stays a
	written-out line until something asks for a second one. */
	b.WriteString(`
	WriteLine(line);

	/* What the server's frames cost while that was happening

	Its own line, because it is about the machine rather than about the bots, and it should be
	possible to read a run's frame times without parsing everything else. */
	char perf[ENGINEER_LINE_LENGTH];
	FormatEx(perf, sizeof(perf),
		"{\"event\":\"perf\",\"map\":\"%s\",\"wave\":%d,\"frames\":%d,"
		... "\"frames_slow\":%d,\"frames_stalled\":%d,\"frame_mean_ms\":%.2f,\"frame_worst_ms\":%.1f,"
		... "\"red\":%d}",
		g_sMap, g_iWave, g_Wave.frames, g_Wave.framesSlow, g_Wave.framesStalled,
		g_Wave.frames > 0 ? g_Wave.frameTotalMs / float(g_Wave.frames) : 0.0,
		g_Wave.frameWorstMs, CountTeam(TFTeam_Red, false));

	WriteLine(perf);

	WriteEngineers("end");

	g_flWaveStart = 0.0;
}
`)
	b.WriteString(spFeaturesFired())
	return []byte(b.String())
}

// spFeaturesFired is the stats plugin's half of the fired counters: the natives
// the bots plugin registers, declared optional so this plugin loads without it,
// and the difference since the last wave line as name:count pairs.
func spFeaturesFired() string {
	return fmt.Sprintf(`
/* Which features ran during the wave, and how many times each answered true
 *
 * Read from the bots plugin, which counts since it loaded. The difference since the last line
 * written is what the wave gets, so the break before it counts too: a feature that only fires
 * between waves is a feature that fired. */
native int %[1]s();
native int %[2]s(int id);
native int %[3]s(int id, char[] name, int maxlen);

#define FEATURES_FIRED_MAX 64

static int g_iFeaturesFiredAtLastLine[FEATURES_FIRED_MAX];
static bool g_bHasFeatureNatives;

// From AskPluginLoad2: the plugin has to load on a server without the mod.
void FeaturesFiredMarkOptional()
{
	MarkNativeAsOptional(%[1]q);
	MarkNativeAsOptional(%[2]q);
	MarkNativeAsOptional(%[3]q);
}

// From OnAllPluginsLoaded.
void FeaturesFiredFind()
{
	g_bHasFeatureNatives = GetFeatureStatus(FeatureType_Native, %[2]q) == FeatureStatus_Available
		&& GetFeatureStatus(FeatureType_Native, %[1]q) == FeatureStatus_Available
		&& GetFeatureStatus(FeatureType_Native, %[3]q) == FeatureStatus_Available;
}

void FeaturesFiredSince(char[] out, int maxlen)
{
	out[0] = '\0';
	if (!g_bHasFeatureNatives)
		return;

	int count = %[1]s();
	if (count > FEATURES_FIRED_MAX)
		count = FEATURES_FIRED_MAX;

	for (int id = 0; id < count; id++)
	{
		int fired = %[2]s(id);
		int since = fired - g_iFeaturesFiredAtLastLine[id];
		g_iFeaturesFiredAtLastLine[id] = fired;
		if (since <= 0)
			continue;

		char name[64];
		%[3]s(id, name, sizeof(name));
		if (out[0] != '\0')
			StrCat(out, maxlen, ",");

		Format(out, maxlen, "%%s%%s:%%d", out, name, since);
	}
}
`, FeatureCountNative, FeatureFiredNative, FeatureNameNative)
}

// spFormatLines is the escaped format string, already quoted and broken up.
func spFormatLines() []string {
	var lines []string
	var cur strings.Builder

	cur.WriteString(`"{`)
	for i, f := range WaveRecord {
		piece := spEscape(`"`+f.JSON+`":`) + spFieldValue(f)
		if i < len(WaveRecord)-1 {
			piece += ","
		} else {
			piece += "}"
		}

		if cur.Len() > 1 && cur.Len()+len(piece) > spLineWidth {
			lines = append(lines, cur.String()+`"`)
			cur.Reset()
			cur.WriteString(`"`)
		}
		cur.WriteString(piece)
	}

	return append(lines, cur.String()+`"`)
}

func spFieldValue(f WaveField) string {
	if f.Literal != "" {
		return spEscape(`"` + f.Literal + `"`)
	}
	if strings.HasPrefix(f.Verb, `"`) {
		return spEscape(f.Verb)
	}
	return f.Verb
}

// spEscape quotes for a SourcePawn string literal. The fields are lowercase
// identifiers and format verbs, so the double quote is the only escape needed.
func spEscape(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }
