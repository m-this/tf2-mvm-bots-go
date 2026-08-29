package spgen

import (
	"fmt"
	"strconv"
	"strings"
)

// SourcePawn emits the decision graph as data: three parallel arrays of nodes
// and one array of roots indexed by the axes. It declares no function and
// calls nothing, so it belongs in the file the differential test compiles.
func (t Table) SourcePawn(cfg Config) string {
	var b strings.Builder
	p := cfg.Prefix

	b.WriteString("/* The decision as data, so the edge can ask lazily\n")
	b.WriteString(" *\n")
	b.WriteString(" * Three of the predicates have side effects, so filling them all in\n")
	b.WriteString(" * before deciding is not what the plugin does today. A node names one\n")
	b.WriteString(" * predicate and the node to go to for each answer; the edge asks for a\n")
	b.WriteString(" * predicate only when the walk reaches it, and never twice, because the\n")
	b.WriteString(" * answer is what chose the next node.\n")
	b.WriteString(" *\n")
	b.WriteString(" * A node with " + p + "NodePredicate of -1 answers: " + p + "NodeWhenTrue\n")
	b.WriteString(" * holds the outcome.\n")
	b.WriteString(" */\n")

	fmt.Fprintf(&b, "#define %sPredicateCount %d\n\n", p, len(t.Lazy.Predicates))
	b.WriteString("enum\n{\n")
	for i, name := range t.Lazy.Predicates {
		comma := ","
		if i == len(t.Lazy.Predicates)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "\t%sPred%s = %d%s\n", p, name, i, comma)
	}
	b.WriteString("};\n\n")

	fmt.Fprintf(&b, "#define %sNodeCount %d\n\n", p, len(t.Predicate))
	writeCells(&b, p+"NodePredicate", p+"NodeCount", t.Predicate)
	writeCells(&b, p+"NodeWhenTrue", p+"NodeCount", t.WhenTrue)
	writeCells(&b, p+"NodeWhenFalse", p+"NodeCount", t.WhenFalse)

	dims := ""
	for _, a := range t.Lazy.Axes {
		dims += "[" + strconv.Itoa(a.Size) + "]"
	}
	fmt.Fprintf(&b, "int %sRoot%s = %s;\n", p, dims, nested(t.Roots, sizes(t.Lazy.Axes), 0))
	return b.String()
}

func sizes(axes []Axis) []int {
	out := make([]int, 0, len(axes))
	for _, a := range axes {
		out = append(out, a.Size)
	}
	return out
}

// nested writes a row-major slice as the nested braces SourcePawn wants.
func nested(cells []int32, dims []int, depth int) string {
	if len(dims) == 1 {
		parts := make([]string, 0, len(cells))
		for _, c := range cells {
			parts = append(parts, strconv.Itoa(int(c)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	stride := 1
	for _, d := range dims[1:] {
		stride *= d
	}
	pad := strings.Repeat("\t", depth+1)
	parts := make([]string, 0, dims[0])
	for i := range dims[0] {
		parts = append(parts, pad+nested(cells[i*stride:(i+1)*stride], dims[1:], depth+1))
	}
	return "{\n" + strings.Join(parts, ",\n") + "\n" + strings.Repeat("\t", depth) + "}"
}

// writeCells wraps at eight per line, which keeps a table of a few hundred
// nodes readable in a diff.
func writeCells(b *strings.Builder, name, count string, cells []int32) {
	fmt.Fprintf(b, "int %s[%s] = {\n", name, count)
	for i := 0; i < len(cells); i += 8 {
		end := min(i+8, len(cells))
		parts := make([]string, 0, 8)
		for _, c := range cells[i:end] {
			parts = append(parts, strconv.Itoa(int(c)))
		}
		line := "\t" + strings.Join(parts, ", ")
		if end < len(cells) {
			line += ","
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("};\n\n")
}
