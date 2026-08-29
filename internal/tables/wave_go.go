package tables

import (
	"fmt"
	"strings"
)

// GoWaveParser is the struct the reports unmarshal a wave line into, with one
// tagged field per entry in the table. It is the other end of the writer
// SourcePawnWaveWriter emits, and neither side can be renamed alone.
func GoWaveParser(pkg string) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "// Code generated from internal/tables/wave.go. DO NOT EDIT.\n\npackage %s\n\n", pkg)
	fmt.Fprintf(&b, "// Event is the value of the event field on a wave line. A file holds other\n"+
		"// events too, so a reader that does not filter on this is reading setup rows\n"+
		"// and engineer rows as waves.\nconst Event = %q\n\n", WaveEvent)

	b.WriteString("// Wave is one wave, as the statistics plugin wrote it.\ntype Wave struct {\n")

	width := 0
	for _, f := range WaveRecord {
		width = max(width, len(f.GoName()))
	}

	for _, f := range WaveRecord {
		fmt.Fprintf(&b, "\t%-*s %-7s `json:%q`\n", width, f.GoName(), f.GoType(), f.JSON)
	}

	b.WriteString("}\n")
	return []byte(b.String())
}
