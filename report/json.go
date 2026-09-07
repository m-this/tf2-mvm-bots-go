package main

import (
	runs "github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

/*
The report's numbers, for a reader that is not a person.

The prose report is the one to read. This is the same figures in one object, so
a comparison can be checked by something other than eyes: the fifty-five scripts
that read a results file across these sessions were mostly asking for a number
the report had already computed and printed with words around it. See mvm-1ro.
*/
type Summary struct {
	File string `json:"file"`
	// Run is what produced the file, when the file says.
	Run *runs.Run `json:"run,omitempty"`

	Waves   int `json:"waves"`
	Cleared int `json:"cleared"`

	RobotKills int `json:"robot_kills"`
	GiantKills int `json:"giant_kills"`
	Deaths     int `json:"defender_deaths"`
	Backstabs  int `json:"backstabs"`

	SentriesLost     int `json:"sentries_lost"`
	BuildingRepaired int `json:"building_repaired"`
	BuildingDamage   int `json:"building_damage"`
	Busters          int `json:"buster_detonations"`

	Damage       int `json:"damage"`
	TankDamage   int `json:"tank_damage"`
	SentryDamage int `json:"sentry_damage"`
	Healing      int `json:"healing"`
	Ubers        int `json:"ubers"`

	DamageBy     map[string]int `json:"damage_by_class"`
	KillsBy      map[string]int `json:"kills_by_class"`
	GiantsBy     map[string]int `json:"giants_by_class"`
	KilledBy     map[string]int `json:"killed_by"`
	CauseBy      map[string]int `json:"death_cause"`
	HealingBy    map[string]int `json:"healing_by_class"`
	SelfDamageBy map[string]int `json:"self_damage_by_class"`
	SelfDeathsBy map[string]int `json:"self_deaths_by_class"`
}

func asSummary(path string, s summary) Summary {
	out := Summary{
		File: path, Waves: s.waves, Cleared: s.cleared,
		RobotKills: s.kills, GiantKills: s.giants, Deaths: s.deaths, Backstabs: s.stabs,
		SentriesLost: s.sentriesLost, BuildingRepaired: s.repaired,
		BuildingDamage: s.buildingDamage, Busters: s.busters,
		Damage: s.damage, TankDamage: s.tankDamage, SentryDamage: s.sentryDamage,
		Healing: s.healing, Ubers: s.ubers,
		DamageBy: s.damageBy, KillsBy: s.killsBy, GiantsBy: s.giantsBy,
		KilledBy: s.killedBy, CauseBy: s.causeBy, HealingBy: s.healingBy,
		SelfDamageBy: s.selfDamageBy, SelfDeathsBy: s.selfDeathsBy,
	}
	if run, found, err := runs.ReadRun(path); err == nil && found {
		out.Run = &run
	}
	return out
}

// Comparison is two files read against each other, the first being the one
// under test, which is the order the prose report uses too.
type Comparison struct {
	After  Summary `json:"after"`
	Before Summary `json:"before"`
	// NotCompared is why the two may not be read against each other, when they
	// may not: a different machine is the usual reason.
	NotCompared string `json:"not_compared,omitempty"`
}
