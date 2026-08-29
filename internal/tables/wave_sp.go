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
 * that nearly lost it, and a change that clears the same waves faster is a change that worked */
static void WriteWaveResult(const char[] result)
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

	b.WriteString("\n\tWriteLine(line);\n}\n")
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
