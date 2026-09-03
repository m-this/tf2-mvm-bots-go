package body_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

/*
TestAPluginExternIsWorkAndNotADependency keeps the port's own list of work
honest.

//sp:plugin means the plugin has this in hand-written SourcePawn and the port
has not reached it. There were 34 of them and 28 named somebody else's include
-- SourceMod's own stocks, stocklib, tf2utils -- which is not work and never
will be: reimplementing a dependency in Go is vendoring it by hand rather than
calling it. Those say //sp:library now, and the count that is left is the real
one.

The check is that the name is not declared under plugin/testbed/build/src, which
is where the vendored includes are fetched to.
*/
func TestAPluginExternIsWorkAndNotADependency(t *testing.T) {
	root := plugin.SkipOrFail(t)

	declared, err := spbody.ExternsFromDir("../engine")
	if err != nil {
		t.Fatalf("reading the extern declarations: %v", err)
	}
	vendored, err := vendoredNames(filepath.Join(root, "testbed", "build", "src"))
	if err != nil {
		t.Skipf("the vendored includes are not built: %v", err)
	}
	for qualified, x := range declared.Funcs {
		if !x.Plugin || !vendored[x.Func] {
			continue
		}
		t.Errorf("%s says //sp:plugin and %s is declared by a vendored include; say //sp:library, which is not work",
			qualified, x.Func)
	}
}

// vendoredNames are the functions the fetched includes declare, read off the
// stock and native lines because that is what a declaration looks like there.
func vendoredNames(dir string) (map[string]bool, error) {
	var files []string
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if ext := filepath.Ext(path); ext == ".inc" || ext == ".sp" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, path := range files {
		text, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for line := range strings.Lines(string(text)) {
			fields := strings.Fields(line)
			if len(fields) < 2 || !slices.Contains([]string{"stock", "native"}, fields[0]) {
				continue
			}
			name, _, isCall := strings.Cut(strings.Join(fields[1:], " "), "(")
			if !isCall {
				continue
			}
			if words := strings.Fields(name); len(words) > 0 {
				out[words[len(words)-1]] = true
			}
		}
	}
	return out, nil
}
