package bindgen

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/bindings"
)

// The three shapes a name can take in the plugin. A free call is `Foo(`, a
// member is `x.Foo(` or `x.Foo =`, and a definition is the plugin declaring
// the thing itself.
var (
	freeCall   = regexp.MustCompile(`(^|[^.\w])([A-Z][A-Za-z0-9_]*)\s*\(`)
	memberUse  = regexp.MustCompile(`\.([A-Za-z_][A-Za-z0-9_]*)\s*(\(|=[^=])`)
	definition = regexp.MustCompile(`(?m)^\s*(?:public |static |stock |native |forward )*[A-Za-z_][A-Za-z0-9_\[\]]*\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

// UnboundReport is the edge of what can be generated: the names the plugin
// uses that the generated package does not declare. Everything on these two
// lists has to be reached some other way, so their length is the size of the
// hand-written remainder.
type UnboundReport struct {
	// Natives are free call sites with no declaration in the tree and no
	// definition in the plugin.
	Natives []string
	// Members are `x.Name` uses whose Name the package declares on no type.
	Members []string
	// Calls and MemberUses are the totals the two lists are drawn from.
	Calls, MemberUses int
}

// Unbound compares what the plugin uses against what the package declares.
//
// It is a name comparison, not a type check: SourcePawn is not parsed at the
// call site, so a name the package declares counts as bound even if the call
// passes it the wrong arguments. That makes the numbers a floor on what still
// has to be written by hand, not a ceiling.
func Unbound(pluginRoot string, res *Result) (*UnboundReport, error) {
	sources, err := pluginSources(pluginRoot)
	if err != nil {
		return nil, err
	}
	calls, members := map[string]bool{}, map[string]bool{}
	local := map[string]bool{}
	for _, path := range sources {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("bindgen: reading %s: %w", path, err)
		}
		// Comments and string literals are scanned out first: the plugin
		// passes VScript source as strings and keeps commented-out engine
		// prototypes beside the code, and both read as call sites.
		code := string(bindings.Code(src))
		collect(calls, freeCall.FindAllStringSubmatch(code, -1), 2)
		collect(members, memberUse.FindAllStringSubmatch(code, -1), 1)
		collect(local, definition.FindAllStringSubmatch(code, -1), 1)
		addLocalDeclarations(local, bindings.Parse(path, src))
	}
	rep := &UnboundReport{Calls: len(calls), MemberUses: len(members)}
	for name := range calls {
		if !local[name] && !res.Declared(name) && !res.Declared(name+"_") && !res.Declared("New"+name) {
			rep.Natives = append(rep.Natives, name)
		}
	}
	for name := range members {
		if !local[name] && !res.declaredMember(name) {
			rep.Members = append(rep.Members, name)
		}
	}
	slices.Sort(rep.Natives)
	slices.Sort(rep.Members)
	return rep, nil
}

func collect(into map[string]bool, matches [][]string, group int) {
	for _, m := range matches {
		into[m[group]] = true
	}
}

// addLocalDeclarations folds in the names the plugin declares as declarations
// rather than as function definitions: its own methodmaps, enum structs,
// enums and #defines. The parser reads a .sp the same way it reads an
// include, so this needs no second syntax.
func addLocalDeclarations(local map[string]bool, f *bindings.File) {
	maps.Copy(local, declaredTypes(f))
	for _, mm := range f.Methodmaps {
		for _, m := range mm.Methods {
			local[m.Name] = true
		}
		for _, pr := range mm.Properties {
			local[pr.Name] = true
		}
	}
	for _, es := range f.EnumStructs {
		for _, m := range es.Methods {
			local[m.Name] = true
		}
		for _, fl := range es.Fields {
			local[fl.Name] = true
		}
	}
	for _, en := range f.Enums {
		for _, entry := range en.Entries {
			local[entry.Name] = true
		}
	}
	for _, d := range f.Defines {
		local[d.Name] = true
	}
	for _, n := range slices.Concat(f.Natives, f.Stocks) {
		local[n.Name] = true
	}
}

// pluginSources lists the plugin's own SourcePawn, sorted.
func pluginSources(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(p, ".sp") || strings.HasSuffix(p, ".inc")) {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("bindgen: walking %s: %w", root, err)
	}
	slices.Sort(paths)
	return paths, nil
}
