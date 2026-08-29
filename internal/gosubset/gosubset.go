// Package gosubset refuses every Go construct the SourcePawn body generator
// cannot translate, naming the construct and what to write instead.
//
// The rule set descends from pass1_illegal_code.go of SourceGo
// (github.com/Nirari-Technologies/Go2SourcePawn), MIT, Copyright (c) 2020
// Kevin Yonan. See ATTRIBUTION.md for what was taken and what was not.
package gosubset

import (
	"cmp"
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

// CheckFiles reports every construct outside the subset across one package's
// files, in file then source order. Package-level types and functions are
// collected from every file before any file is checked, so a declaration in
// one file is known to all the others.
func CheckFiles(fset *token.FileSet, files []*ast.File, cfg Config) []Diagnostic {
	c := newChecker(fset, cfg)
	for _, f := range files {
		c.collect(f)
	}
	for _, f := range files {
		c.beginFile()
		c.checkFile(f)
	}
	slices.SortStableFunc(c.diags, func(a, b Diagnostic) int {
		return cmp.Or(strings.Compare(a.Pos.Filename, b.Pos.Filename), a.Pos.Offset-b.Pos.Offset)
	})
	return c.diags
}

// CheckFile reports every construct in f outside the subset, in source order.
//
// It knows only the package-level names f itself declares. A file that calls a
// function or names a type declared in another file of the same package is
// refused as unknown, which is correct for a single-file body and wrong for a
// package. Check a package with CheckFiles or CheckDir.
func CheckFile(fset *token.FileSet, f *ast.File, cfg Config) []Diagnostic {
	return CheckFiles(fset, []*ast.File{f}, cfg)
}

// CheckSource parses src and checks it as a single file, with the same
// single-file limit as CheckFile. Parse errors are reported as refusals so a
// caller has one failure mode rather than two.
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

// CheckDir checks every non-test .go file directly under dir as one package,
// so a type or function declared in one file is known to all of them.
func CheckDir(dir string, cfg Config) ([]Diagnostic, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
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
		if len(files) > 0 && f.Name.Name != files[0].Name.Name {
			return nil, fmt.Errorf("%s declares package %s, but %s declares package %s: one directory is one package",
				path, f.Name.Name, fset.Position(files[0].Package).Filename, files[0].Name.Name)
		}
		files = append(files, f)
	}
	return CheckFiles(fset, files, cfg), nil
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
