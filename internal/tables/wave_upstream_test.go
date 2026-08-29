package tables_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

var reWaveKey = regexp.MustCompile(`"([a-z0-9_]+)":("?%[-.0-9]*[dfs]"?|"wave_end")`)

// parseWaveWriter reads the wave_end FormatEx back out of a SourcePawn file:
// the field names in the order the format string writes them, each paired with
// the argument that fills it.
//
// It is deliberately picky. A format string with more placeholders than
// arguments is the bug this table exists to stop, so a mismatch is a failure
// here rather than a shrug.
func parseWaveWriter(t *testing.T, src string) []tables.WaveField {
	t.Helper()

	start := strings.Index(src, `"{\"event\":\"wave_end\"`)
	if start < 0 {
		t.Fatal("no wave_end format string")
	}

	body := src[start:]
	end := strings.Index(body, "}\",\n")
	if end < 0 {
		t.Fatal("wave_end format string does not end")
	}

	format := unescapeSP(joinSPLiteral(body[:end+2]))
	args := splitSPArgs(body[end+3:])

	matches := reWaveKey.FindAllStringSubmatch(format, -1)
	if len(matches) == 0 {
		t.Fatal("no fields in the wave_end format string")
	}

	var out []tables.WaveField
	next := 0
	for _, m := range matches {
		if m[2] == `"wave_end"` {
			out = append(out, tables.WaveField{JSON: m[1], Literal: "wave_end"})
			continue
		}
		if next >= len(args) {
			t.Fatalf("field %q has no argument: %d placeholders, %d arguments", m[1], len(matches)-1, len(args))
		}
		out = append(out, tables.WaveField{JSON: m[1], Verb: m[2], SP: args[next]})
		next++
	}

	if next != len(args) {
		t.Fatalf("%d arguments for %d placeholders", len(args), next)
	}
	return out
}

// joinSPLiteral glues the `...` continuations of one string literal together.
func joinSPLiteral(block string) string {
	var b strings.Builder
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "..."))
		line = strings.TrimSuffix(strings.TrimPrefix(line, `"`), `"`)
		b.WriteString(line)
	}
	return b.String()
}

func unescapeSP(s string) string { return strings.ReplaceAll(s, `\"`, `"`) }

// splitSPArgs splits the argument list on the commas that are not inside a call
// or an index, and drops the whitespace the plugin wraps them with.
func splitSPArgs(block string) []string {
	var out []string
	var cur strings.Builder

	depth := 0
	for _, r := range block {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth == 0 {
				out = append(out, spArg(cur.String()))
				return out
			}
			depth--
		case ',':
			if depth == 0 {
				out = append(out, spArg(cur.String()))
				cur.Reset()
				continue
			}
		}
		cur.WriteRune(r)
	}
	return append(out, spArg(cur.String()))
}

func spArg(s string) string { return strings.Join(strings.Fields(s), "") }

// TestWaveTableRoundTrips is the proof for the wave record: every field
// mvmbots_stats.sp writes today comes back out of the Go table with the same
// name, the same verb and the same argument, in the same order.
func TestWaveTableRoundTrips(t *testing.T) {
	t.Parallel()

	upstream := parseWaveWriter(t, readUpstream(t, "testbed", "stats", "mvmbots_stats.sp"))

	if len(upstream) != len(tables.WaveRecord) {
		t.Fatalf("mvmbots_stats.sp writes %d fields, the table has %d", len(upstream), len(tables.WaveRecord))
	}

	for i, want := range upstream {
		got := tables.WaveRecord[i]
		if got != want {
			t.Errorf("field %d:\n got %+v\nwant %+v", i, got, want)
		}
	}
}

// TestGeneratedWaveWriterRoundTrips runs the generated SourcePawn back through
// the same parser, so the line breaking and the escaping have to survive too.
func TestGeneratedWaveWriterRoundTrips(t *testing.T) {
	t.Parallel()

	upstream := parseWaveWriter(t, readUpstream(t, "testbed", "stats", "mvmbots_stats.sp"))
	generated := parseWaveWriter(t, string(tables.SourcePawnWaveWriter()))

	if len(generated) != len(upstream) {
		t.Fatalf("generated %d fields, mvmbots_stats.sp writes %d", len(generated), len(upstream))
	}

	for i := range upstream {
		if generated[i] != upstream[i] {
			t.Errorf("field %d:\n got %+v\nwant %+v", i, generated[i], upstream[i])
		}
	}
}

// TestGeneratedParserReadsEveryWrittenField parses the generated Go and checks
// its tags against the table. This is the join the plugin never had: the writer
// and the reader are compared rather than hoped about.
func TestGeneratedParserReadsEveryWrittenField(t *testing.T) {
	t.Parallel()

	tags := parseGoTags(t, tables.GoWaveParser("waveline"), "Wave")

	if len(tags) != len(tables.WaveRecord) {
		t.Fatalf("generated parser has %d fields, the table has %d", len(tags), len(tables.WaveRecord))
	}

	for i, f := range tables.WaveRecord {
		if tags[i] != f.JSON {
			t.Errorf("field %d: parser reads %q, the writer writes %q", i, tags[i], f.JSON)
		}
	}
}

// TestReportFieldsAreInTheTable. testbed/report reads a subset today; every tag
// it asks for has to be a field something writes, or it reads a zero forever.
func TestReportFieldsAreInTheTable(t *testing.T) {
	t.Parallel()

	known := map[string]bool{}
	for _, f := range tables.WaveRecord {
		known[f.JSON] = true
	}

	for _, tag := range parseGoTags(t, []byte(readUpstream(t, "testbed", "report", "main.go")), "wave") {
		if tag == "-" || tag == "" {
			continue
		}
		if !known[tag] {
			t.Errorf("testbed/report reads %q, which nothing in the wave line writes", tag)
		}
	}
}

// TestWaveFieldNamesAreUnique. Two fields with one name is one field, and the
// second value silently wins on the reading side.
func TestWaveFieldNamesAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	goNames := map[string]int{}
	for i, f := range tables.WaveRecord {
		if first, ok := seen[f.JSON]; ok {
			t.Errorf("field %q at index %d and %d", f.JSON, first, i)
		}
		seen[f.JSON] = i

		if first, ok := goNames[f.GoName()]; ok {
			t.Errorf("Go name %q for %q at index %d and for index %d", f.GoName(), f.JSON, i, first)
		}
		goNames[f.GoName()] = i
	}
}

// parseGoTags is the json tag of every field of one struct, in order.
func parseGoTags(t *testing.T, src []byte, typeName string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "src.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing generated Go: %v", err)
	}

	var out []string
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != typeName {
			return true
		}
		structType, ok := spec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		found = true

		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				t.Fatalf("tag %s: %v", field.Tag.Value, err)
			}
			name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
			out = append(out, name)
		}
		return false
	})

	if !found {
		t.Fatalf("no struct %s", typeName)
	}
	return out
}

/*
	TestGeneratedWaveWriterIsTheWholeShippedFunction

The round trip above proves the fields. It says nothing about the rest of the
function, and the rest of the function is where the generator was wrong: it
stopped at WriteLine(line) and dropped the perf line, the engineer lines and the
reset of g_flWaveStart. Adopting that file would have taken the frame times out
of every run and let the next round reset write a row of zeros.

So this compares the whole of WriteWaveResult with the whole of the shipped one,
with comments and line breaks normalised away, because those are the two things
the generator is allowed to move.
*/
func TestGeneratedWaveWriterIsTheWholeShippedFunction(t *testing.T) {
	t.Parallel()

	shipped, ok := waveResultFunc(readUpstream(t, "testbed", "stats", "mvmbots_stats.sp"))
	if !ok {
		t.Fatal("no WriteWaveResult in the shipped mvmbots_stats.sp")
	}
	generated, ok := waveResultFunc(string(tables.SourcePawnWaveWriter()))
	if !ok {
		t.Fatal("no WriteWaveResult in the generated wave writer")
	}
	if shipped != generated {
		t.Errorf("the generated WriteWaveResult is not the shipped one:\nshipped:   %s",
			firstDifference(shipped, generated))
	}
}

// waveResultFunc pulls WriteWaveResult out of a file and normalises it: the
// adjacent string literals FormatEx sees as one are joined, comments go, and
// runs of whitespace become one space.
func waveResultFunc(src string) (string, bool) {
	i := strings.Index(src, "static void WriteWaveResult")
	if i < 0 {
		return "", false
	}
	j := strings.Index(src[i:], "\n}\n")
	if j < 0 {
		return "", false
	}
	body := src[i : i+j+3]
	body = reContinuation.ReplaceAllString(body, "")
	body = reBlockComment.ReplaceAllString(body, "")
	body = reLineComment.ReplaceAllString(body, "")
	return strings.TrimSpace(reSpaceRun.ReplaceAllString(body, " ")), true
}

var (
	reContinuation = regexp.MustCompile(`"\s*\n\s*\.\.\.\s*"`)
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLineComment  = regexp.MustCompile(`//[^\n]*`)
	reSpaceRun     = regexp.MustCompile(`\s+`)
)

// firstDifference names where two normalised functions part company, because a
// diff of two thousand characters on one line is not a report anybody reads.
func firstDifference(a, b string) string {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return fmt.Sprintf("%q\n%s%q", around(a, i), strings.Repeat(" ", 11), around(b, i))
		}
	}
	if len(a) == len(b) {
		return "no difference"
	}
	longer, at := a, len(b)
	if len(b) > len(a) {
		longer, at = b, len(a)
	}
	return fmt.Sprintf("one ends where the other continues, at %d: %q", at, around(longer, at))
}

func around(s string, i int) string {
	return s[max(0, i-70):min(len(s), i+70)]
}
