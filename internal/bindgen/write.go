package bindgen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write puts the generated package in dir, replacing whatever was there. A
// generated tree is disposable: it is rebuilt from the includes and never
// edited, so clearing it first is what keeps a stale file from surviving a
// rename.
func Write(dir string, r *Result) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("bindgen: clearing %s: %w", dir, err)
	}
	// The output is source: read by people, diffed by git and compiled by the
	// Go toolchain. 0755 and 0644 are what source is, and it is the same pair
	// cmd/gen writes its own generated files with.
	//nolint:gosec // G301: a generated source directory is world-readable on purpose.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("bindgen: making %s: %w", dir, err)
	}
	for name, body := range r.Files {
		//nolint:gosec // G306: a generated source file is world-readable on purpose.
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return fmt.Errorf("bindgen: writing %s: %w", name, err)
		}
	}
	return nil
}
