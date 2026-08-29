/*
mapview draws a test-bed run as a picture of the map it was played on.

	go run ./cmd/mapview -results ../tf2-mvm-bots/results/run-plain-1.jsonl -out /tmp/run

One PNG per wave, named for the map and the wave. The nav mesh is the floor
plan and the run is drawn over it: one line per bot, coloured by class, with a
cross where he ended up, and buildings in white.

The mesh is found by map name under internal/navmesh/testdata, which carries
the seven MvM maps. -nav names one directly for anything else.

The picture carries no text, because drawing text needs a font and a font is a
dependency. The legend is printed here instead.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
	"github.com/m-this/tf2-mvm-bots-go/internal/runmap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		results = flag.String("results", "", "the run's JSONL, as the statistics plugin wrote it")
		nav     = flag.String("nav", "", "a .nav or .nav.gz file; found by map name when empty")
		out     = flag.String("out", ".", "directory to write the PNGs into")
		size    = flag.Int("size", runmap.DefaultSize, "longest side of the picture, in pixels")
	)
	flag.Parse()

	if *results == "" {
		return fmt.Errorf("mapview: -results is required")
	}

	parsed, err := runmap.ReadFile(*results)
	if err != nil {
		return err
	}

	if len(parsed.Waves) == 0 {
		return fmt.Errorf("mapview: no bot samples in %s, so there is nothing to draw", *results)
	}

	path := *nav
	if path == "" {
		if parsed.Map == "" {
			return fmt.Errorf("mapview: %s names no map, so -nav has to say which mesh to use", *results)
		}
		path = filepath.Join("internal", "navmesh", "testdata", parsed.Map+".nav.gz")
	}

	mesh, err := navmesh.LoadFile(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*out, 0o750); err != nil {
		return fmt.Errorf("mapview: %w", err)
	}

	for _, wave := range parsed.Waves {
		name := fmt.Sprintf("%s-wave%d.png", parsed.Map, wave.Number)
		if err := write(filepath.Join(*out, name), mesh, wave, *size); err != nil {
			return err
		}

		fmt.Printf("%s: %d bots, %d buildings\n", name, len(wave.Tracks), len(wave.Buildings))
		for _, class := range wave.Classes() {
			colour, known := runmap.ClassColour(class)
			note := ""
			if !known {
				note = " (no colour of its own)"
			}
			fmt.Printf("    %-9s #%02X%02X%02X%s\n", class, colour.R, colour.G, colour.B, note)
		}
	}

	return nil
}

func write(path string, mesh *navmesh.Mesh, wave runmap.Wave, size int) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("mapview: %w", err)
	}

	if err := runmap.PNG(file, runmap.Draw(mesh, wave, size)); err != nil {
		// The close is still owed, and its error is the less interesting of the
		// two: a picture that failed to encode is not saved by a clean close.
		_ = file.Close()

		return fmt.Errorf("mapview: writing %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("mapview: writing %s: %w", path, err)
	}

	return nil
}
