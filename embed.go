/*
Package mvmbots carries the generator's own inputs.

cmd/gen translates the Go under internal/body into SourcePawn, which means it
reads Go source at run time. That works in a checkout and not anywhere else, and
"anywhere else" is the point: tf2-archipelago pins this module with a tool
directive in its go.mod and runs the generator from the module cache, where
internal/body is not a directory anybody can open.

So the sources travel with the binary. The embedded copy is written to a
temporary directory when the real one is not there, and the generator reads that
instead. Nothing chooses between them by configuration: the directory is either
under the working directory or it is not.

Only what the body and action generators read is embedded: internal/action,
internal/body and internal/engine. The tables and the decisions are compiled in
already, being ordinary Go.
*/
package mvmbots

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed internal/action internal/body internal/engine go.mod
var sources embed.FS

// SourceRoot is a directory holding internal/body and internal/engine, and the
// cleanup for it. It is the working directory when this is a checkout, and an
// unpacked copy of the embedded sources when it is a module cache.
func SourceRoot(dir string) (root string, done func(), err error) {
	if _, err := os.Stat(filepath.Join(dir, "internal", "engine")); err == nil {
		return dir, func() {}, nil
	}

	out, err := os.MkdirTemp("", "mvmbots-sources")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(out) }

	err = fs.WalkDir(sources, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(out, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		body, err := sources.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o600)
	})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return out, cleanup, nil
}
