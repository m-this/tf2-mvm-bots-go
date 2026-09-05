// Command gen writes every generated file. Nothing it writes is committed, and
// nothing it writes is edited by hand: make check regenerates and fails if the
// output moved.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	mvmbots "github.com/m-this/tf2-mvm-bots-go"
	"github.com/m-this/tf2-mvm-bots-go/internal/adopt"
	"github.com/m-this/tf2-mvm-bots-go/internal/bindgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/generated"
)

func main() {
	out := flag.String("out", "gen", "directory to write generated files into")
	plugin := flag.String("plugin", "plugin", "the plugin tree, read for the include tree")
	adoptFiles := flag.Bool("adopt", false, "also write the adopted files into the plugin tree")
	flag.Parse()

	if err := run(*out, *plugin, *adoptFiles); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

/*
	The include tree the bindings are generated from

It lives inside the plugin's test-bed build directory because that is what
build.sh already downloads and caches. Bindings generated from includes older
than the compiler are bindings for the wrong API, which is mvm-z83.21.
*/
func includeRoot(plugin string) string {
	return filepath.Join(plugin, "testbed", "build")
}

// writeBindings emits the SourceMod API as Go. Absent includes are not an
// error: a fresh clone has not run build.sh yet, and the rest of the output
// does not depend on them.
func writeBindings(out, plugin string) error {
	root := includeRoot(plugin)
	if !isDir(root) {
		fmt.Fprintf(os.Stderr, "gen: no include tree at %s, skipping bindings\n", root)
		return nil
	}

	res, err := bindgen.Generate(bindgen.Options{Root: root, Package: "sm"})
	if err != nil {
		return fmt.Errorf("generating bindings: %w", err)
	}
	dir := filepath.Join(out, "go", "sm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	if err := bindgen.Write(dir, res); err != nil {
		return fmt.Errorf("writing bindings: %w", err)
	}
	fmt.Fprintf(os.Stderr, "gen: %d binding files, %d refusals\n", len(res.Files), len(res.Refusals))
	return nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func run(out, plugin string, adoptFiles bool) error {
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clearing %s: %w", out, err)
	}
	// The bodies are generated from Go source, which is in the working
	// directory in a checkout and in the binary anywhere else.
	sources, done, err := mvmbots.SourceRoot(".")
	if err != nil {
		return fmt.Errorf("resolving the generator's own sources: %w", err)
	}
	defer done()

	emitted, err := generated.Files(sources)
	if err != nil {
		return err
	}
	for name, body := range emitted {
		path := filepath.Join(out, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("making %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	if err := writeBindings(out, plugin); err != nil {
		return err
	}
	if !adoptFiles {
		return nil
	}
	return writeAdopted(sources, plugin)
}

// writeAdopted refreshes the generated files the plugin tree commits, so the
// drift test has nothing to report.
func writeAdopted(sources, plugin string) error {
	files, err := adopt.Files(sources)
	if err != nil {
		return err
	}
	moved, err := adopt.Write(plugin, files)
	if err != nil {
		return err
	}
	for _, name := range moved {
		fmt.Fprintf(os.Stderr, "gen: adopted %s\n", name)
	}
	return nil
}
