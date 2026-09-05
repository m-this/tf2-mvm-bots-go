package tables_test

import (
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

/*
	TestGeneratedWaveWriterRoundTrips

The generated FormatEx read back by the parser: the field names in the order the
format string writes them, each paired with the argument that fills it, have to
be the table. The line breaking and the escaping are the generator's, so they
have to survive it.
*/
func TestGeneratedWaveWriterRoundTrips(t *testing.T) {
	t.Parallel()

	generated := parseWaveWriter(t, string(tables.SourcePawnWaveWriter()))

	if len(generated) != len(tables.WaveRecord) {
		t.Fatalf("the generated writer writes %d fields, the table has %d", len(generated), len(tables.WaveRecord))
	}

	for i, want := range tables.WaveRecord {
		if got := generated[i]; got != want {
			t.Errorf("field %d:\n got %+v\nwant %+v", i, got, want)
		}
	}
}

// TestGeneratedParserReadsEveryWrittenField parses the generated Go and checks
// its tags against the table. This is the join the plugin never had: the writer
// and the reader are compared rather than hoped about.
func TestGeneratedParserReadsEveryWrittenField(t *testing.T) {
	t.Parallel()

	tags := parseGoTags(t, tables.GoWaveParser("waveline"), "Record")

	if len(tags) != len(tables.WaveRecord) {
		t.Fatalf("generated parser has %d fields, the table has %d", len(tags), len(tables.WaveRecord))
	}

	for i, f := range tables.WaveRecord {
		if tags[i] != f.JSON {
			t.Errorf("field %d: parser reads %q, the writer writes %q", i, tags[i], f.JSON)
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
