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
	return []byte(b.String())
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
