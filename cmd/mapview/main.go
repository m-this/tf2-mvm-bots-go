/*
mapview draws a test-bed run as a picture of the map it was played on.

	go run ./cmd/mapview -results results/run-plain-1.jsonl -out /tmp/run

One PNG per wave, named for the map and the wave. The nav mesh is the floor
plan and the run is drawn over it: one line per bot, coloured by class, with a
cross where he ended up, and buildings in white.

The mesh is found by map name under internal/navmesh/testdata, which carries
the seven MvM maps. -nav names one directly for anything else.

A question about one spot is asked through a window:

	go run ./cmd/mapview -map mvm_bigrock -window "-1200 3700 200 4800" -spots -falls

Without -results this draws the map alone. -spots lays the map config's spots
over it as hollow diamonds, and -falls marks every edge whose drop costs
health, red where it kills. A pixel is then a few units, which is the scale a
nest and the rock beside it are apart.

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
	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
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
		results = flag.String("results", "", "the run's JSONL, as the statistics plugin wrote it; empty draws the map alone")
		mapName = flag.String("map", "", "the map, when there is no run to read it from")
		nav     = flag.String("nav", "", "a .nav or .nav.gz file; found by map name when empty")
		out     = flag.String("out", ".", "directory to write the PNGs into")
		size    = flag.Int("size", runmap.DefaultSize, "longest side of the picture, in pixels")
		window  = flag.String("window", "", "the part of the map to draw, as \"x0 y0 x1 y1\" in map units; empty for all of it")
		spots   = flag.Bool("spots", false, "lay the map config's spots over the picture")
		falls   = flag.Bool("falls", false, "mark every edge whose drop hurts")
	)
	flag.Parse()

	if *results == "" && *mapName == "" {
		return fmt.Errorf("mapview: -results or -map is required")
	}

	parsed := runmap.Run{Map: *mapName}
	if *results != "" {
		var err error
		if parsed, err = runmap.ReadFile(*results); err != nil {
			return err
		}
		if len(parsed.Waves) == 0 {
			return fmt.Errorf("mapview: no bot samples in %s, so there is nothing to draw", *results)
		}
	}
	if parsed.Map == "" {
		return fmt.Errorf("mapview: %s names no map, so -map has to say which", *results)
	}

	path := *nav
	if path == "" {
		path = filepath.Join("internal", "navmesh", "testdata", parsed.Map+".nav.gz")
	}

	mesh, err := navmesh.LoadFile(path)
	if err != nil {
		return err
	}

	view := runmap.View{Size: *size}
	if view.Window, err = parseWindow(*window); err != nil {
		return err
	}
	if *falls {
		view.Falls = runmap.HurtingFalls(mesh)
	}
	if *spots {
		if view.Spots, err = configSpots(parsed.Map); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(*out, 0o750); err != nil {
		return fmt.Errorf("mapview: %w", err)
	}

	waves := parsed.Waves
	if len(waves) == 0 {
		waves = []runmap.Wave{{Map: parsed.Map}}
	}

	for _, wave := range waves {
		name := fmt.Sprintf("%s-wave%d.png", parsed.Map, wave.Number)
		if wave.Number == 0 {
			name = parsed.Map + "-map.png"
		}
		if err := write(filepath.Join(*out, name), mesh, wave, view); err != nil {
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

	printLegend(view)

	return nil
}

func printLegend(view runmap.View) {
	seen := map[navmesh.SpotKind]bool{}
	for _, s := range view.Spots {
		if seen[s.Kind] {
			continue
		}
		seen[s.Kind] = true
		c := runmap.SpotColour(s.Kind)
		fmt.Printf("    %-14s #%02X%02X%02X diamond\n", s.Kind, c.R, c.G, c.B)
	}
	for _, s := range view.Spots {
		fmt.Printf("    %s\n", s)
	}
	if len(view.Falls) > 0 {
		fmt.Printf("    %d hurting falls, red dots, brighter where the drop kills\n", len(view.Falls))
	}
}

func parseWindow(s string) (runmap.Window, error) {
	var w runmap.Window
	if s == "" {
		return w, nil
	}

	n, err := fmt.Sscanf(s, "%f %f %f %f", &w.MinX, &w.MinY, &w.MaxX, &w.MaxY)
	if err != nil || n != 4 {
		return w, fmt.Errorf("mapview: -window wants \"x0 y0 x1 y1\", got %q", s)
	}
	if w.MinX > w.MaxX {
		w.MinX, w.MaxX = w.MaxX, w.MinX
	}
	if w.MinY > w.MaxY {
		w.MinY, w.MaxY = w.MaxY, w.MinY
	}

	return w, nil
}

func configSpots(mapName string) ([]navmesh.Spot, error) {
	path, err := plugin.Path("configs", "defenderbots", "map", mapName+".cfg")
	if err != nil {
		return nil, err
	}

	config, err := navmesh.LoadMapConfig(path)
	if err != nil {
		return nil, err
	}

	return config.Spots, nil
}

func write(path string, mesh *navmesh.Mesh, wave runmap.Wave, view runmap.View) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("mapview: %w", err)
	}

	if err := runmap.PNG(file, runmap.DrawView(mesh, wave, view)); err != nil {
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
