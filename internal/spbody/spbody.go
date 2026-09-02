package spbody

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config names what one emission may reach and what it calls the result.
type Config struct {
	// Prefix goes in front of every emitted identifier, because Action,
	// Address and RoundState are SourceMod's names already.
	Prefix string
	// Externs are the functions the body calls and this package does not
	// translate, by the qualified name the body writes: "engine.GetClientTeam".
	Externs map[string]Extern
	// Tags are the types a body names and this package does not declare,
	// by the qualified Go name: "engine.Class" is SourcePawn's TFClassType.
	Tags map[string]string
	// Declare overrides the emitted declaration line of a named function,
	// which is how a callback the engine calls gets the signature the engine
	// calls it with rather than the one derived from the Go. The body is
	// translated the same way either way.
	Declare map[string]string
	// Import maps an import path to the identifier the body writes it as,
	// so a selector can be resolved back to an extern without go/types
	// having to name the package twice.
	Import map[string]string
}

// Extern is one call the engine already has, and how the emitted SourcePawn
// reaches it.
type Extern struct {
	// Func is the SourcePawn function the call site names: the native
	// itself, or SDKCall, or LoadFromAddress.
	Func string
	// Lead are arguments written before the Go ones. A native has none; an
	// SDKCall has the handle prepared at load.
	Lead []string
	// Plugin says this is not the engine at all but a function the plugin
	// still has in hand-written SourcePawn. Every one of these is work the
	// port has not done, and the day the port does it the extern has to go:
	// a function owned in both places is the duplication this epic exists
	// to remove.
	Plugin bool
	// ReturnsArray says the SourcePawn declaration returns the array rather
	// than filling a parameter with it, which is the float[] form. Such a
	// call can only be an argument to something else: spcomp will not assign
	// one to a sized array, so this package will not emit that either.
	ReturnsArray bool
	/*
		Borrowed says the handle this returns belongs to somebody else.

		SourceMod's menus are the case: new Menu(handler) hands the menu
		to SourceMod, which keeps it alive until the player is done and
		then fires MenuAction_End, where the handler deletes it. So the
		function that built it must not close it, and the ownership rule
		has to be told rather than guess.
	*/
	Borrowed bool
	// Global says this is not a call at all: SourceMod's MaxClients is a
	// variable, and the Go declaration is a function only because the
	// subset has no other way to name something it does not own.
	Global bool
	// Method says the call is written on a receiver: SourceMod's API is
	// methodmaps, and myBot.GetVisionInterface() has no plain function
	// behind it to call instead.
	Method bool
	// Slot says the extern is a plugin array indexed by its first argument,
	// not a call: m_flRepathTime[actor]. Set says the same array written to,
	// which takes the value as its second argument.
	Slot bool
	Set  bool
	// InPlace says the call writes its answer back into its first argument,
	// which is how SourcePawn spells ScaleVector. Nothing is appended, and
	// the assignment has to name that same argument.
	InPlace bool
	// Trail are arguments written after the Go ones and after the result
	// buffer, which is the only place GetAngleVectors will take the two
	// vectors a caller does not want.
	Trail []string
	// Cast says the call is SourcePawn's view_as, which is a tag change
	// written around the value rather than a call at all.
	Cast bool
	// Choice says the extern is SourcePawn's ?: and not a call at all. Go
	// has no conditional expression, and a two-branch if writing a string
	// is not a thing the emitter can spell: SourcePawn holds text in a
	// sized buffer, so that shape becomes two copies where the plugin
	// wrote one argument.
	Choice bool
	// Delete says the method is SourcePawn's delete on its receiver, which
	// makes the receiver's type a handle: a lifetime the emitter tracks and
	// refuses to leave open.
	Delete bool
	// Property says the method is written without parentheses, which is what
	// SourcePawn calls a property: convar.BoolValue.
	Property bool
	// Sized says the buffer this fills is followed by its length, which is
	// how a call that reads something into a buffer is declared:
	// GetEntityClassname(entity, buffer, maxlen).
	Sized bool
	// Fills says the buffer and its length come first instead, which is how
	// a call that writes text is declared: Format(buffer, maxlen, ...).
	Fills bool
	// Body says the opposite of Plugin: this names SourcePawn the port
	// already generates, in another package. The emitted SourcePawn is one
	// flat namespace, so calling it is calling it by name; what this buys is
	// the assertion, because internal/body requires a body of that name to
	// exist and refuses the declaration when it does not.
	Body bool
}

// Generated is one emission: the SourcePawn, and what was left out of it.
type Generated struct {
	Package string
	// Emitted are the SourcePawn names this emission declares, so a caller
	// can hold them against what is still declared as an extern.
	Emitted []string
	// Source is the bodies: plain functions, which run anywhere a
	// SourcePawn VM does once the engine calls are stubbed.
	Source string
	// Hooks is the DHook callbacks. They are a separate file because they
	// name DHookParam and MRESReturn, which live in SourceMod's includes
	// and not in the standalone SourcePawn the differential test runs
	// under. Empty when the package declares no hook.
	Hooks string
	// Skipped are the files that were parsed for names but not translated,
	// because they are generated rather than authored.
	Skipped []string
}

// GenerateDir type checks every non-test .go file directly under dir as one
// package and translates it.
//
// A parse error, a type error and a construct with no SourcePawn are all the
// same failure here: an error naming file, line and what to write instead. The
// caller has nothing to sort out because nothing partial is returned.
func GenerateDir(dir string, cfg Config) (Generated, error) {
	fset := token.NewFileSet()
	files, skipped, err := parseDir(fset, dir)
	if err != nil {
		return Generated{}, err
	}
	if len(files) == 0 {
		return Generated{}, fmt.Errorf("spbody: %s holds no Go file to translate", dir)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	imp, err := newModuleImporter(fset, dir)
	if err != nil {
		return Generated{}, err
	}
	conf := types.Config{Importer: imp}
	pkg, err := conf.Check(dir, fset, files, info)
	if err != nil {
		return Generated{}, fmt.Errorf("spbody: %s does not type check: %w", dir, err)
	}
	e := &emitter{fset: fset, cfg: cfg, info: info, pkg: pkg}
	e.run(files)
	if err := e.err(); err != nil {
		return Generated{}, err
	}
	const banner = "/* Generated by internal/spbody from the Go it is named after. Do not edit. */\n\n"
	g := Generated{Package: pkg.Name(), Emitted: e.emitted, Source: banner + e.prologue() + e.b.String(), Skipped: skipped}
	if e.hooks.Len() > 0 {
		g.Hooks = banner + e.hooks.String()
	}
	return g, nil
}

// generatedMarker is the line the Go toolchain defines as "this file was not
// written by a person". A file carrying it in a body directory is the extern
// seam or a table: it is read for its names and never translated.
const generatedMarker = "// Code generated "

func parseDir(fset *token.FileSet, dir string) ([]*ast.File, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("spbody: reading %s: %w", dir, err)
	}
	var files []*ast.File
	var skipped []string
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, fmt.Errorf("spbody: parsing %s: %w", path, err)
		}
		if isGenerated(f) {
			skipped = append(skipped, name)
		}
		files = append(files, f)
	}
	return files, skipped, nil
}

func isGenerated(f *ast.File) bool {
	for _, group := range f.Comments {
		if group.Pos() > f.Package {
			return false
		}
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, generatedMarker) && strings.HasSuffix(c.Text, "DO NOT EDIT.") {
				return true
			}
		}
	}
	return false
}
