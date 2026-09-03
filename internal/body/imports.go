package body

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

/*
A body may import another body

The emitted SourcePawn is one flat namespace, so a body calling another body is
a call by name. What used to stop it was that the caller had no way to learn the
name: the emitter resolves a qualified call through Config.Externs, and the only
package in there was internal/engine. Sharing a decision therefore meant
declaring an extern for it by hand, restating in internal/engine a signature
that already existed in Go a directory away, and that cost was paid again by
every behaviour that wanted the same thing.

The registry already knows what it needs to remove that cost. It knows every
generated package, and each package's declarations already carry the SourcePawn
name they are emitted under. So the table below is read off the registry rather
than typed, and it does for an import what a hand-written extern did for a call:
it says what the SourcePawn is called.

An extern is still the only way to reach the engine, and still the only way to
reach SourcePawn the port has not written. Neither of those is in the registry,
which is the point.
*/

// imported is one generated package a body may import: the path it is imported
// by, and what it offers.
type imported struct {
	Path    string
	Package string
	Exports []spbody.Export
}

// importable is every generated package, by import path. Reading it parses each
// registered directory once and type checks none of them: a name is all an
// import needs, and the caller's own type check resolves the package itself.
func importable(root string) (map[string]imported, error) {
	out := make(map[string]imported, len(All)+len(Actions))
	for _, b := range slices.Concat(All, Actions) {
		dir := filepath.Join(root, b.Dir)
		exports, err := spbody.Exports(dir, b.Prefix)
		if err != nil {
			return nil, err
		}
		name, err := packageName(dir)
		if err != nil {
			return nil, err
		}
		importPath := modulePath + "/" + b.Dir
		if _, twice := out[importPath]; twice {
			return nil, fmt.Errorf("%s is in the registry twice", b.Dir)
		}
		out[importPath] = imported{Path: importPath, Package: name, Exports: exports}
	}
	return out, nil
}

/*
externsFor is what one package's own imports resolve to.

A cross-package call comes out as an ordinary body extern, because that is
exactly what it is: SourcePawn this port generates, in another file, reached by
name. Making it one means the emitter needs no new case -- externOf already
resolves a qualified call through this map -- and the ownership rule keeps
applying to it unchanged.

Only the packages this one actually imports go in. Putting every generated
package in every emission would let a body call one it never imported, which Go
would refuse and the generator would not, and the two disagreeing is the whole
class of bug the subset checker exists to prevent.
*/
func externsFor(root, dir string, base map[string]spbody.Extern, table map[string]imported) (map[string]spbody.Extern, error) {
	imports, err := importsOf(filepath.Join(root, dir))
	if err != nil {
		return nil, err
	}
	var wanted []imported
	for _, p := range imports {
		if p == modulePath+"/"+ExternDir || !strings.HasPrefix(p, modulePath+"/") {
			continue
		}
		other, generated := table[p]
		if !generated {
			return nil, fmt.Errorf("%s imports %s, which the registry does not generate; add it to internal/body.All, or do not import it", dir, p)
		}
		if p == modulePath+"/"+filepath.ToSlash(dir) {
			return nil, fmt.Errorf("%s imports itself", dir)
		}
		wanted = append(wanted, other)
	}
	if len(wanted) == 0 {
		return base, nil
	}
	out := make(map[string]spbody.Extern, len(base)+len(wanted)*8)
	for k, v := range base {
		out[k] = v
	}
	for _, other := range wanted {
		for _, e := range other.Exports {
			if e.Const {
				// Folded where it is written, so there is
				// nothing to resolve and nothing to name.
				continue
			}
			key := other.Package + "." + e.Go
			if _, taken := out[key]; taken {
				return nil, fmt.Errorf("%s imports %s, whose package name %s is already an extern package", dir, other.Path, other.Package)
			}
			out[key] = spbody.Extern{Func: e.SP, Body: true}
		}
	}
	return out, nil
}

// subsetPackages is what gosubset accepts a body importing: each generated
// package by the exported names it offers, so an import of one is checked
// against what it actually declares rather than waved through.
func subsetPackages(table map[string]imported) map[string][]string {
	out := make(map[string][]string, len(table))
	for p, other := range table {
		names := make([]string, 0, len(other.Exports))
		for _, e := range other.Exports {
			names = append(names, e.Go)
		}
		out[p] = names
	}
	return out
}

// importsOf is the module-local import paths of every non-test file in dir.
func importsOf(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("body: reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return nil, fmt.Errorf("body: parsing %s: %w", name, err)
		}
		for _, spec := range f.Imports {
			out = append(out, strings.Trim(spec.Path.Value, `"`))
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

// packageName is what the directory's files call themselves, which is the
// identifier a caller writes in front of the dot.
func packageName(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("body: reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.PackageClauseOnly)
		if err != nil {
			return "", fmt.Errorf("body: parsing %s: %w", name, err)
		}
		return f.Name.Name, nil
	}
	return "", fmt.Errorf("body: %s holds no Go file", dir)
}
