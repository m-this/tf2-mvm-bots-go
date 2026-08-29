// Package gosubset refuses every Go construct the SourcePawn body generator
// cannot translate, naming the construct and what to write instead.
//
// The rule set descends from pass1_illegal_code.go of SourceGo
// (github.com/Nirari-Technologies/Go2SourcePawn), MIT, Copyright (c) 2020
// Kevin Yonan. See ATTRIBUTION.md for what was taken and what was not.
package gosubset

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// maxDiagnostics stops a file of prose from producing a report nobody reads.
const maxDiagnostics = 200

// Diagnostic is one refusal: where it is, what was written, what to write.
type Diagnostic struct {
	Pos       token.Position
	Construct string
	Fix       string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s. %s", d.Pos, d.Construct, d.Fix)
}

// Config names what a body may reach outside its own declarations.
type Config struct {
	// Natives are the generated native bindings a body may call.
	Natives []string
	// Packages maps an importable path to the identifiers a body may use
	// from it. Anything not listed is refused.
	Packages map[string][]string
}

// DefaultConfig allows the float helpers that the real decision code uses and
// nothing else. math.Abs and math.Min map onto FloatAbs and MinFloat.
func DefaultConfig() Config {
	return Config{
		Packages: map[string][]string{
			"math": {"Abs", "Sqrt", "Floor", "Ceil", "Pow", "Sin", "Cos", "Atan2", "Pi", "MaxFloat32"},
		},
	}
}

// CheckFile reports every construct in f outside the subset, in source order.
func CheckFile(fset *token.FileSet, f *ast.File, cfg Config) []Diagnostic {
	c := newChecker(fset, cfg)
	c.collect(f)
	c.checkFile(f)
	slices.SortStableFunc(c.diags, func(a, b Diagnostic) int {
		return a.Pos.Offset - b.Pos.Offset
	})
	return c.diags
}

// CheckSource parses src and checks it. Parse errors are reported as refusals
// so a caller has one failure mode rather than two.
func CheckSource(filename, src string, cfg Config) []Diagnostic {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return []Diagnostic{{
			Pos:       token.Position{Filename: filename},
			Construct: "the file does not parse as Go: " + err.Error(),
			Fix:       "fix the syntax; the subset is checked over a parsed file",
		}}
	}
	return CheckFile(fset, f, cfg)
}

// CheckDir checks every non-test .go file directly under dir.
func CheckDir(dir string, cfg Config) ([]Diagnostic, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	var diags []Diagnostic
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		diags = append(diags, CheckFile(fset, f, cfg)...)
	}
	return diags, nil
}

// Join turns a refusal list into one error, or nil when nothing was refused.
func Join(diags []Diagnostic) error {
	if len(diags) == 0 {
		return nil
	}
	lines := make([]string, 0, len(diags))
	for _, d := range diags {
		lines = append(lines, d.Error())
	}
	return fmt.Errorf("%d construct(s) outside the Go subset:\n%s", len(diags), strings.Join(lines, "\n"))
}
