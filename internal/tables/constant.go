package tables

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/action/engineerbuildteleporter"
	"github.com/m-this/tf2-mvm-bots-go/internal/action/engineeridle"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/pathing"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/shared"
)

/*
	The constants whose relation to another constant is the thing that matters

mvm-1pq was TELEPORTER_EXIT_RADIUS at 150 against BUSTER_BLAST_RANGE at 400: the
teleporter exit was built at well under half the reach of the one robot whose job
is to detonate on the nest, so a single buster took the sentry and the team's
forward spawn together. Neither number is wrong on its own. The relation between
them was wrong, and it was written down nowhere, so nothing could notice.

A constant with no stated relation to the constant it has to respect is two facts
written down once each and never compared. The relations below are that
comparison, and TestRelationsHold runs it over the values the bodies declare.

The values are not copied here. Each one is read from the Go package that emits
it, so the table cannot say one thing while the plugin ships another.
*/

// Constant is one plugin #define this table has an opinion about.
type Constant struct {
	// Name is the SourcePawn name, which is what a relation reads.
	Name string
	// Value is the Go constant the body declares under that name.
	Value float64
	// Unit is what the number counts: units, seconds, paths per frame.
	Unit string
	Why  string
}

// Constants is every constant a relation below names. A constant with no
// relation does not belong here: the body's own comment is a better home for
// it than a second copy in Go.
var Constants = []Constant{
	{
		Name:  "BUSTER_BLAST_RANGE",
		Value: shared.BusterBlastRange,
		Unit:  "units",
		Why:   "How far a sentry buster's detonation reaches. Everything the engineer builds is placed against it.",
	},
	{
		Name:  "TELEPORTER_EXIT_RADIUS",
		Value: engineerbuildteleporter.ExitRadius,
		Unit:  "units",
		Why:   "The tight ring, and a build-validity radius: far enough out not to be inside his own sentry. It answers where the exit can go, never where it should.",
	},
	{
		Name:  "TELEPORTER_EXIT_RADIUS_SAFE",
		Value: engineerbuildteleporter.ExitRadiusSafe,
		Unit:  "units",
		Why:   "The ring tried first, sized off the blast rather than off the build reach.",
	},
	{
		Name:  "NEST_RELOCATE_HAUL_TIME",
		Value: engineeridle.RelocateHaulTime,
		Unit:  "seconds",
		Why:   "How long the engineer gets to move a sentry to better ground before he puts it down where he stands.",
	},
	{
		Name:  "CARRY_GIVE_UP_TIME",
		Value: engineeridle.CarryGiveUpTime,
		Unit:  "seconds",
		Why:   "How long he may walk around holding a building at all, whatever he picked it up for.",
	},
	{
		Name:  "PATHS_PER_FRAME",
		Value: pathing.PathsPerFrame,
		Unit:  "paths per frame",
		Why:   "How many paths the whole team may compute in one frame. NavAreaBuildPath is what the watchdog caught the server inside three times.",
	},
	{
		Name:  "PATH_REFRESH_INTERVAL",
		Value: pathing.PathRefreshInterval,
		Unit:  "seconds",
		Why:   "How often a bot with a path asks for it again. It is the team's demand for paths.",
	},
}

// Value is one constant by its SourcePawn name, and whether the table has it.
func Value(name string) (float64, bool) {
	for _, c := range Constants {
		if c.Name == name {
			return c.Value, true
		}
	}
	return 0, false
}
