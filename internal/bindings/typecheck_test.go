package bindings

import (
	"go/ast"
	"go/importer"
	goparser "go/parser"
	gotoken "go/token"
	"go/types"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// undeclared matches the only type-check error a correctly emitted file may
// still produce: a tag declared by another include.
var undeclared = regexp.MustCompile(`undefined: ([A-Za-z_][A-Za-z0-9_]*)`)

// TestEmittedGoTypeChecks proves the emitter's output is real Go, not just
// text that happens to format. The only errors tolerated are references to
// types another include declares; those are listed, because they are the
// cross-file edges a generation driver has to satisfy.
func TestEmittedGoTypeChecks(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := ParseFile(filepath.Join(includeRoot(), tc.include))
			if err != nil {
				t.Skipf("include not present: %v", err)
			}
			out, err := Emit(f, Options{Package: "sp"})
			if err != nil {
				t.Fatalf("emitting: %v", err)
			}
			missing := typeCheck(t, tc.name, out.Source)
			t.Logf("%s needs %d types from other includes: %s",
				tc.name, len(missing), strings.Join(missing, " "))
		})
	}
}

func typeCheck(t *testing.T, name string, src []byte) []string {
	t.Helper()
	fset := gotoken.NewFileSet()
	file, err := goparser.ParseFile(fset, name+".go", src, goparser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing emitted Go: %v", err)
	}
	seen := map[string]bool{}
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			m := undeclared.FindStringSubmatch(err.Error())
			if m == nil {
				t.Errorf("emitted Go does not type-check: %v", err)
				return
			}
			seen[m[1]] = true
		},
	}
	_, _ = conf.Check("sp", fset, []*ast.File{file}, nil)
	missing := make([]string, 0, len(seen))
	for k := range seen {
		missing = append(missing, k)
	}
	sort.Strings(missing)
	return missing
}
