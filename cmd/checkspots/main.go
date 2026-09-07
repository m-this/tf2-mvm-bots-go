// Command checkspots says which dispenser spot each authored nest would take,
// for every map config.
//
//	go run ./cmd/checkspots
//
// A nest that falls back on every map in the file is a nest whose spots were
// authored for a different pairing. It reads the configs and nothing else, so
// it costs nothing and needs no server. It was testbed/checkspots.py, which is
// the last Python the test-bed shipped: see mvm-qbi.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-this/tf2-mvm-bots-go/internal/navmesh"
	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "checkspots: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("configs", "", "the map config directory, empty for the plugin tree's own")
	flag.Parse()

	from := *dir
	if from == "" {
		root, err := plugin.Dir()
		if err != nil {
			return err
		}
		from = filepath.Join(root, "configs", "defenderbots", "map")
	}

	configs, err := navmesh.LoadMapConfigs(from)
	if err != nil {
		return err
	}
	for _, c := range configs {
		if report := navmesh.DispenserReport(c); report != "" {
			fmt.Print(report)
		}
	}
	return nil
}
