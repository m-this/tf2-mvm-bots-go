package tables

/*
	The constants whose relation to another constant is the thing that matters

mvm-1pq was TELEPORTER_EXIT_RADIUS at 150 against BUSTER_BLAST_RANGE at 400: the
teleporter exit was built at well under half the reach of the one robot whose job
is to detonate on the nest, so a single buster took the sentry and the team's
forward spawn together. Neither number is wrong on its own. The relation between
them was wrong, and it was written down nowhere, so nothing could notice.

A constant with no stated relation to the constant it has to respect is two facts
written down once each and never compared. The relations below are that
comparison, checked against the values read out of the plugin at the pin.

They are not emitted. These constants live in three different plugin files with
the reasoning written above each one, and moving them here would trade a
comprehension win for a churn cost that mvm-z83.16 has not paid yet.
*/

// Constant is one plugin #define this table has an opinion about.
type Constant struct {
	Name string
	// File is where it is defined, relative to source/redbots3.
	File string
	// Unit is what the number counts: units, seconds, paths per frame.
	Unit string
	Why  string
}

// Constants is every constant a relation below names. A constant with no
// relation does not belong here: the plugin's own comment is a better home for
// it than a second copy in Go.
var Constants = []Constant{
	{
		Name: "BUSTER_BLAST_RANGE",
		File: "util.sp",
		Unit: "units",
		Why:  "How far a sentry buster's detonation reaches. Everything the engineer builds is placed against it.",
	},
	{
		Name: "TELEPORTER_EXIT_RADIUS",
		File: "behavior/engineerbuildteleporter.sp",
		Unit: "units",
		Why:  "The tight ring, and a build-validity radius: far enough out not to be inside his own sentry. It answers where the exit can go, never where it should.",
	},
	{
		Name: "TELEPORTER_EXIT_RADIUS_SAFE",
		File: "behavior/engineerbuildteleporter.sp",
		Unit: "units",
		Why:  "The ring tried first, sized off the blast rather than off the build reach.",
	},
	{
		Name: "NEST_RELOCATE_HAUL_TIME",
		File: "behavior/engineeridle.sp",
		Unit: "seconds",
		Why:  "How long the engineer gets to move a sentry to better ground before he puts it down where he stands.",
	},
	{
		Name: "CARRY_GIVE_UP_TIME",
		File: "behavior/engineeridle.sp",
		Unit: "seconds",
		Why:  "How long he may walk around holding a building at all, whatever he picked it up for.",
	},
	{
		Name: "PATHS_PER_FRAME",
		File: "nextbot_behavior.sp",
		Unit: "paths per frame",
		Why:  "How many paths the whole team may compute in one frame. NavAreaBuildPath is what the watchdog caught the server inside three times.",
	},
}
