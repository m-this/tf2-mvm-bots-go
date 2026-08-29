package spbody

import (
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

/*
	moduleImporter resolves the one import a body is allowed

The extern package, which is in this module, is type checked from its source,
because the body compiles against it and a stale export file would type check
against last week's declarations. Anything else is the standard library, and
comes from the installed export data.

x/tools/go/packages does this and more, and this is the whole of what is needed:
one module, source files on disk, no build tags and no cgo.
*/
type moduleImporter struct {
	fset    *token.FileSet
	module  string // the module path from go.mod
	root    string // the directory holding go.mod
	std     types.Importer
	done    map[string]*types.Package
	pending map[string]bool
}

func newModuleImporter(fset *token.FileSet, dir string) (*moduleImporter, error) {
	root, module, err := moduleRoot(dir)
	if err != nil {
		return nil, err
	}
	return &moduleImporter{
		fset:    fset,
		module:  module,
		root:    root,
		std:     importer.Default(),
		done:    make(map[string]*types.Package),
		pending: make(map[string]bool),
	}, nil
}

func (m *moduleImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := m.done[path]; ok {
		return pkg, nil
	}
	if !strings.HasPrefix(path, m.module+"/") {
		return m.std.Import(path)
	}
	if m.pending[path] {
		return nil, fmt.Errorf("spbody: %s imports itself", path)
	}
	m.pending[path] = true
	defer delete(m.pending, path)

	dir := filepath.Join(m.root, filepath.FromSlash(strings.TrimPrefix(path, m.module+"/")))
	files, _, err := parseDir(m.fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("spbody: %s holds no Go file", dir)
	}
	conf := types.Config{Importer: m}
	pkg, err := conf.Check(path, m.fset, files, nil)
	if err != nil {
		return nil, fmt.Errorf("spbody: %s does not type check: %w", dir, err)
	}
	m.done[path] = pkg
	return pkg, nil
}

func moduleRoot(dir string) (root, module string, err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	for at := abs; ; {
		text, err := os.ReadFile(filepath.Join(at, "go.mod"))
		if err == nil {
			path, ok := modulePath(string(text))
			if !ok {
				return "", "", fmt.Errorf("spbody: %s/go.mod declares no module path", at)
			}
			return at, path, nil
		}
		parent := filepath.Dir(at)
		if parent == at {
			return "", "", fmt.Errorf("spbody: no go.mod at or above %s", abs)
		}
		at = parent
	}
}

func modulePath(text string) (string, bool) {
	for line := range strings.Lines(text) {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], true
		}
	}
	return "", false
}
