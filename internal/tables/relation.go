package tables

import "fmt"

/*
	Relation is one thing that has to stay true between two constants

Statement is what it says, in the plugin's own names, so a failure reads without
opening anything. Holds is the same thing the compiler can run, taking a lookup
rather than the values directly because a relation naming a constant that no
longer exists has to fail loudly rather than compare against a zero.

Why is the bug. A relation with no bug behind it is a rule somebody invented,
and those get worked around rather than respected.
*/
type Relation struct {
	Name      string
	Statement string
	Why       string
	// Uses is every constant Holds reads, checked against Constants before it
	// runs.
	Uses  []string
	Holds func(value func(string) float64) bool
}

// TickRate is the server's tick, which is not a constant in the plugin because
// it is not the plugin's to choose. It is here because a budget per frame only
// means something per second.
const TickRate = 66.0

/*
	PathRefreshInterval is the 0.2 second repath at nextbot_behavior.sp:703

An inline literal, not a #define, which is why the relation below could not be
written against a name. PATHS_PER_FRAME is sized against it in prose and against
nothing in code.
*/
const PathRefreshInterval = 0.2

// RedTeamSize is the cap on defending bots, which is what turns one bot's
// refresh rate into the team's demand for paths. See mvm-jmo.
const RedTeamSize = 6

// Relations is the whole set. Every one of them has a closed or open bug behind
// it, named in Why.
var Relations = []Relation{
	{
		Name:      "the exit stands outside the blast",
		Statement: "TELEPORTER_EXIT_RADIUS_SAFE >= BUSTER_BLAST_RANGE",
		Why:       "mvm-1pq: the exit sat at 150 against a 400 unit blast, so one buster took the sentry and the forward spawn together.",
		Uses:      []string{"TELEPORTER_EXIT_RADIUS_SAFE", "BUSTER_BLAST_RANGE"},
		Holds: func(v func(string) float64) bool {
			return v("TELEPORTER_EXIT_RADIUS_SAFE") >= v("BUSTER_BLAST_RANGE")
		},
	},
	{
		Name:      "the tight ring is knowingly inside the blast",
		Statement: "TELEPORTER_EXIT_RADIUS < BUSTER_BLAST_RANGE",
		Why:       "mvm-1pq again, from the other side. The tight ring is a fallback for a map with no room at the safe one, so it is meant to be inside the blast. Raising it to look safe would take away the fallback and leave maps with no exit at all.",
		Uses:      []string{"TELEPORTER_EXIT_RADIUS", "BUSTER_BLAST_RANGE"},
		Holds: func(v func(string) float64) bool {
			return v("TELEPORTER_EXIT_RADIUS") < v("BUSTER_BLAST_RANGE")
		},
	},
	{
		Name:      "the carry clock outlasts the haul clock",
		Statement: "CARRY_GIVE_UP_TIME > NEST_RELOCATE_HAUL_TIME",
		Why:       "mvm-1b7: the give-up is the outer bound on holding a building at all. If it fires first the relocation never gets its own timeout, and the engineer puts the sentry down for a reason nobody wrote.",
		Uses:      []string{"CARRY_GIVE_UP_TIME", "NEST_RELOCATE_HAUL_TIME"},
		Holds: func(v func(string) float64) bool {
			return v("CARRY_GIVE_UP_TIME") > v("NEST_RELOCATE_HAUL_TIME")
		},
	},
	{
		Name:      "the path budget covers what the team asks for",
		Statement: fmt.Sprintf("PATHS_PER_FRAME * %g >= %d / %g", TickRate, RedTeamSize, PathRefreshInterval),
		Why:       "mvm-297: two a frame is there to remove the frame where everybody asks at once, not to starve anyone. A budget under the team's own refresh rate would make bots wait for paths, which is a different bug wearing the fix's clothes.",
		Uses:      []string{"PATHS_PER_FRAME"},
		Holds: func(v func(string) float64) bool {
			return v("PATHS_PER_FRAME")*TickRate >= RedTeamSize/PathRefreshInterval
		},
	},
}
