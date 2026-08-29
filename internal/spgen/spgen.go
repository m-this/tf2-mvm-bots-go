// Package spgen turns the gosubset-accepted Go of one package into the
// SourcePawn the plugin includes.
//
// It is the body generator, and it is deliberately smaller than the subset
// gosubset accepts: gosubset says what a body may be written in, spgen says
// what has been implemented, and anything in the gap is refused with a
// position rather than guessed at. The mappings for several results, range
// loops and discarded results are read from SourceGo's pass3, pass9 and
// pass10; see internal/gosubset/ATTRIBUTION.md.
//
// A generated body takes plain values and returns a plain value. It calls no
// native, so the hand-written SourcePawn fills the inputs and maps the result
// onto the real action. That is the design's never-call-back rule and it is
// why this generator needs no binding table.
//
// # What spgen implements, against what gosubset accepts
//
// gosubset is the wider gate and spgen is the narrower one. Everything below
// is accepted by gosubset and refused here, with a position:
//
//   - float64, int64 and uint64, which do not fit a 32-bit cell
//   - several results, which pass3 of SourceGo turns into by-reference
//     parameters and this generator does not emit yet
//   - range loops, which pass9 covers and nothing here needs
//   - any import at all, so a package is self-contained
//   - a conversion between a float and an int, because the rounding is a
//     decision the author owes in Go rather than one a generator guesses
//   - an if or a switch with an init statement, a for with no condition,
//     multiple assignment, and a named type over a basic type with no
//     constants
//
// Names. Every emitted identifier carries Config.Prefix, because Action,
// RoundState and Address are SourceMod's names already; a local whose Go name
// is a SourcePawn keyword gets a trailing underscore.
package spgen

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/gosubset"
)

// Config is what the caller decides: the name every emitted identifier carries
// and the guard the emitted file uses.
type Config struct {
	// Prefix is prepended to every emitted type, constant and function name.
	// It is not optional: Action, RoundState and Plugin_Continue's Action are
	// all SourceMod names already, so an unprefixed emission collides.
	Prefix string
	// Guard is the include guard symbol, without underscores of its own.
	Guard string
}

// Unsupported is one construct spgen has no translation for.
type Unsupported struct {
	Pos       token.Position
	Construct string
	Fix       string
}

func (u Unsupported) Error() string {
	return fmt.Sprintf("%s: %s. %s", u.Pos, u.Construct, u.Fix)
}

// Package is one checked Go package, parsed and type-checked once, ready to
// be emitted as SourcePawn and to be interpreted.
type Package struct {
	fset  *token.FileSet
	info  *types.Info
	pkg   *types.Package
	files []*ast.File
	names []string
}

// Load parses every non-test .go file directly under dir as one package.
//
// gosubset runs first, so a construct outside the subset is refused with the
// subset's own message rather than translated into something plausible.
func Load(dir string) (*Package, error) {
	diags, err := gosubset.CheckDir(dir, gosubset.DefaultConfig())
	if err != nil {
		return nil, err
	}
	if err := gosubset.Join(diags); err != nil {
		return nil, fmt.Errorf("%s is not inside the Go subset: %w", dir, err)
	}

	fset := token.NewFileSet()
	files, names, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("spgen: %s holds no non-test Go file", dir)
	}

	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	conf := types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", "amd64")}
	pkg, err := conf.Check(files[0].Name.Name, fset, files, info)
	if err != nil {
		return nil, fmt.Errorf("spgen: %s does not type-check: %w", dir, err)
	}
	return &Package{fset: fset, info: info, pkg: pkg, files: files, names: names}, nil
}

// SourcePawn emits the package as the SourcePawn the plugin includes.
func (p *Package) SourcePawn(cfg Config) (string, error) {
	if cfg.Prefix == "" {
		return "", fmt.Errorf("spgen: Config.Prefix is empty: every emitted name needs one, or it collides with SourceMod's own Action and RoundState")
	}
	g := &generator{fset: p.fset, info: p.info, pkg: p.pkg, cfg: cfg}
	return g.emit(p.files, p.names)
}

// GenerateDir loads dir and emits it.
func GenerateDir(dir string, cfg Config) (string, error) {
	p, err := Load(dir)
	if err != nil {
		return "", err
	}
	return p.SourcePawn(cfg)
}

// WriteDir generates dir and writes the result to path, unchanged bytes and
// all, so a caller can diff it against what is committed.
func WriteDir(dir, path string, cfg Config) error {
	out, err := GenerateDir(dir, cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil { //nolint:gosec // G306: a generated source file is source, read by people and compiled by spcomp
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func parseDir(fset *token.FileSet, dir string) ([]*ast.File, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)

	files := make([]*ast.File, 0, len(names))
	for _, n := range names {
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing %s: %w", n, err)
		}
		files = append(files, f)
	}
	return files, names, nil
}
