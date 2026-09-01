package bindings

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

func pluginRoot(t *testing.T) string {
	t.Helper()

	return filepath.Join(plugin.SkipOrFail(t), "source")
}

// callSite matches an identifier used as a call. The plugin's own helpers and
// the include declarations share this shape, so the set is a superset of the
// natives it uses.
var callSite = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\s*\(`)

// pluginDefinition matches a function the plugin defines itself. Those names
// need no binding: the call site resolves inside the plugin.
var pluginDefinition = regexp.MustCompile(`(?m)^(?:public |static |stock |native |forward )*[A-Za-z_][A-Za-z0-9_\[\]]*\s+([A-Z][A-Za-z0-9_]*)\s*\(`)

func pluginSources(t *testing.T) []string {
	t.Helper()
	root := pluginRoot(t)
	if _, err := os.Stat(root); err != nil {
		t.Skipf("plugin source not present: %v", err)
	}
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".sp") || strings.HasSuffix(path, ".inc") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking plugin source: %v", err)
	}
	return paths
}

// pluginNames scans the plugin for the names matched by re.
func pluginNames(t *testing.T, re *regexp.Regexp) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, path := range pluginSources(t) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading plugin source: %v", err)
		}
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			names[m[1]] = true
		}
	}
	return names
}

// declared splits every name the tree declares into the free natives and the
// members reachable through a methodmap or enum struct.
func declared(t *testing.T) (natives, members map[string]bool) {
	t.Helper()
	natives, members = map[string]bool{}, map[string]bool{}
	for _, path := range includeFiles(t) {
		f, err := ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range f.Natives {
			natives[n.Name] = true
		}
		for _, n := range f.Stocks {
			natives[n.Name] = true
		}
		for _, mm := range f.Methodmaps {
			members[mm.Name] = true
			for _, m := range mm.Methods {
				members[m.Name] = true
			}
			for _, p := range mm.Properties {
				members[p.Name] = true
			}
		}
		for _, es := range f.EnumStructs {
			members[es.Name] = true
			for _, m := range es.Methods {
				members[m.Name] = true
			}
		}
	}
	return natives, members
}

// TestPluginCallSiteCoverage measures the part of the plugin's call surface
// the parsed includes account for. The rest is the plugin's own SourcePawn
// functions plus the compiler builtins, which have no binding to generate.
func TestPluginCallSiteCoverage(t *testing.T) {
	sites := pluginNames(t, callSite)
	defined := pluginNames(t, pluginDefinition)
	// The plugin declares methodmaps of its own; the parser reads them the
	// same way it reads the includes.
	for _, path := range pluginSources(t) {
		f, err := ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, mm := range f.Methodmaps {
			defined[mm.Name] = true
			for _, m := range mm.Methods {
				defined[m.Name] = true
			}
			for _, pr := range mm.Properties {
				defined[pr.Name] = true
			}
		}
		for _, d := range f.Defines {
			defined[d.Name] = true
		}
	}
	natives, members := declared(t)

	var hitNative, hitMember, local, missed int
	var examples []string
	for name := range sites {
		switch {
		case natives[name]:
			hitNative++
		case members[name]:
			hitMember++
		case defined[name]:
			local++
		default:
			missed++
			if len(examples) < 25 {
				examples = append(examples, name)
			}
		}
	}
	t.Logf("plugin call sites            %d", len(sites))
	t.Logf("resolve to a free native     %d", hitNative)
	t.Logf("resolve to a methodmap member %d", hitMember)
	t.Logf("defined by the plugin itself %d", local)
	t.Logf("no declaration anywhere      %d", missed)
	t.Logf("unresolved sample: %s", strings.Join(examples, " "))

	if hitNative+hitMember == 0 {
		t.Fatal("no plugin call site resolved to a declaration; the parser is not seeing the tree")
	}
	// Everything the plugin calls is either an include declaration or one of
	// its own functions. The residue is a handful of names from a SourceMod
	// newer than the vendored tree; a jump here means the parser regressed.
	if missed > 20 {
		t.Errorf("unaccounted call sites = %d, want at most 20: %v", missed, examples)
	}
}
