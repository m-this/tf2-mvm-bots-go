package tables

import "strings"

// WaveField is one number in the line the statistics plugin writes when a wave
// ends.
//
// JSON is the identity. The SourcePawn format string, the argument it is given
// and the Go struct field that reads it back all come from this one entry, so a
// rename moves all three together. Renaming one of them alone used to read as a
// zero rather than as an error.
type WaveField struct {
	JSON string

	// Literal is a constant value, written with no argument. Only the event
	// name uses it.
	Literal string

	// Verb is the FormatEx placeholder, quoted when the value is a string.
	Verb string

	// SP is the expression handed to FormatEx for this field.
	SP string
}

// GoName is the field name in the generated parser struct.
func (f WaveField) GoName() string {
	var b strings.Builder
	for _, word := range strings.Split(f.JSON, "_") {
		if word == "" {
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return b.String()
}

// GoType is what the verb decodes into.
func (f WaveField) GoType() string {
	switch {
	case f.Literal != "" || f.Verb == `"%s"`:
		return "string"
	case strings.Contains(f.Verb, "f"):
		return "float64"
	default:
		return "int"
	}
}

// WaveEvent is the value of the event field, which is what the readers filter on.
const WaveEvent = "wave_end"

// WaveRecord is the wave line, in the order it is written. Appending a field is
// one entry here and one more counter in the plugin; nothing else moves.
var WaveRecord = []WaveField{
	{JSON: "event", Literal: "wave_end"},
	{JSON: "map", Verb: "\"%s\"", SP: "g_sMap"},
	{JSON: "wave", Verb: "%d", SP: "g_iWave"},
	{JSON: "result", Verb: "\"%s\"", SP: "result"},
	{JSON: "duration", Verb: "%.1f", SP: "duration"},
	{JSON: "robot_kills", Verb: "%d", SP: "g_Wave.robotKills"},
	{JSON: "giant_kills", Verb: "%d", SP: "g_Wave.giantKills"},
	{JSON: "tank_kills", Verb: "%d", SP: "g_Wave.tankKills"},
	{JSON: "sentry_kills", Verb: "%d", SP: "g_Wave.sentryKills"},
	{JSON: "defender_deaths", Verb: "%d", SP: "g_Wave.defenderDeaths"},
	{JSON: "backstabs", Verb: "%d", SP: "g_Wave.backstabs"},
	{JSON: "buster_detonations", Verb: "%d", SP: "g_Wave.busterDetonations"},
	{JSON: "sentries_lost", Verb: "%d", SP: "g_Wave.sentriesLost"},
	{JSON: "dispensers_lost", Verb: "%d", SP: "g_Wave.dispensersLost"},
	{JSON: "upgrades", Verb: "%d", SP: "g_Wave.upgradesBought"},
	{JSON: "upgrade_credits", Verb: "%d", SP: "g_Wave.upgradeCreditsSpent"},
	{JSON: "credits_dropped", Verb: "%d", SP: "g_Wave.creditsDropped"},
	{JSON: "credits_picked_up", Verb: "%d", SP: "g_Wave.creditsAcquired"},
	{JSON: "credits_bonus", Verb: "%d", SP: "g_Wave.creditsBonus"},
	{JSON: "credits_spent", Verb: "%d", SP: "g_Wave.creditsSpent"},
	{JSON: "credits_in_hand", Verb: "%d", SP: "g_Wave.creditsInHand"},
	{JSON: "damage", Verb: "%d", SP: "g_Wave.damageDealt"},
	{JSON: "tank_damage", Verb: "%d", SP: "g_Wave.damageToTanks"},
	{JSON: "sentry_damage", Verb: "%d", SP: "g_Wave.sentryDamage"},
	{JSON: "healing", Verb: "%d", SP: "g_Wave.healingDone"},
	{JSON: "ubers", Verb: "%d", SP: "g_Wave.ubersDeployed"},
	{JSON: "damage_scout", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "damage_sniper", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "damage_soldier", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "damage_demoman", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "damage_medic", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "damage_heavy", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "damage_pyro", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "damage_spy", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "damage_engineer", Verb: "%d", SP: "g_Wave.damageByClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "kills_scout", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "kills_soldier", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "kills_pyro", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "kills_demoman", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "kills_heavy", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "kills_engineer", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "kills_medic", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "kills_sniper", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "kills_spy", Verb: "%d", SP: "g_Wave.killsByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "giantkills_scout", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "giantkills_soldier", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "giantkills_pyro", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "giantkills_demoman", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "giantkills_heavy", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "giantkills_engineer", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "giantkills_medic", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "giantkills_sniper", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "giantkills_spy", Verb: "%d", SP: "g_Wave.giantKillsByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "killedby_scout", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "killedby_soldier", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "killedby_pyro", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "killedby_demoman", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "killedby_heavy", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "killedby_engineer", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "killedby_medic", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "killedby_sniper", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "killedby_spy", Verb: "%d", SP: "g_Wave.deathsToClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "killedby_sentry", Verb: "%d", SP: "g_Wave.deathsToSentry"},
	{JSON: "killedby_tank", Verb: "%d", SP: "g_Wave.deathsToTank"},
	{JSON: "cause_bullet", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_BULLET]"},
	{JSON: "cause_explosion", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_EXPLOSION]"},
	{JSON: "cause_fire", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_FIRE]"},
	{JSON: "cause_melee", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_MELEE]"},
	{JSON: "cause_backstab", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_BACKSTAB]"},
	{JSON: "cause_headshot", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_HEADSHOT]"},
	{JSON: "cause_fall", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_FALL]"},
	{JSON: "cause_other", Verb: "%d", SP: "g_Wave.deathsByCause[DEATH_CAUSE_OTHER]"},
	{JSON: "selfdamage_scout", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "selfdamage_soldier", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "selfdamage_pyro", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "selfdamage_demoman", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "selfdamage_heavy", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "selfdamage_engineer", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "selfdamage_medic", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "selfdamage_sniper", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "selfdamage_spy", Verb: "%d", SP: "g_Wave.selfDamageByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "selfdeaths_scout", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "selfdeaths_soldier", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "selfdeaths_pyro", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "selfdeaths_demoman", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "selfdeaths_heavy", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "selfdeaths_engineer", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Engineer)]"},
	{JSON: "selfdeaths_medic", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "selfdeaths_sniper", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "selfdeaths_spy", Verb: "%d", SP: "g_Wave.selfDeathsByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "demo_pipe_damage", Verb: "%d", SP: "g_Wave.demoPipeDamage"},
	{JSON: "demo_sticky_damage", Verb: "%d", SP: "g_Wave.demoStickyDamage"},
	{JSON: "demo_melee_damage", Verb: "%d", SP: "g_Wave.demoMeleeDamage"},
	{JSON: "soldier_rocket_damage", Verb: "%d", SP: "g_Wave.soldierRocketDamage"},
	{JSON: "soldier_other_damage", Verb: "%d", SP: "g_Wave.soldierOtherDamage"},
	{JSON: "fired_soldier", Verb: "%d", SP: "g_Wave.projectilesFired[view_as<int>(TFClass_Soldier)]"},
	{JSON: "hit_soldier", Verb: "%d", SP: "g_Wave.projectilesHit[view_as<int>(TFClass_Soldier)]"},
	{JSON: "fired_demoman", Verb: "%d", SP: "g_Wave.projectilesFired[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "hit_demoman", Verb: "%d", SP: "g_Wave.projectilesHit[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "jars_thrown", Verb: "%d", SP: "g_Wave.jarsThrown"},
	{JSON: "building_repaired", Verb: "%d", SP: "g_Wave.buildingRepaired"},
	{JSON: "building_damage", Verb: "%d", SP: "g_Wave.buildingDamageTaken"},
	{JSON: "healing_scoreboard", Verb: "%d", SP: "g_Wave.healingScoreboard"},
	{JSON: "healing_scout", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Scout)]"},
	{JSON: "healing_sniper", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Sniper)]"},
	{JSON: "healing_soldier", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Soldier)]"},
	{JSON: "healing_demoman", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_DemoMan)]"},
	{JSON: "healing_medic", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Medic)]"},
	{JSON: "healing_heavy", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Heavy)]"},
	{JSON: "healing_pyro", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Pyro)]"},
	{JSON: "healing_spy", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Spy)]"},
	{JSON: "healing_engineer", Verb: "%d", SP: "g_Wave.healingByClass[view_as<int>(TFClass_Engineer)]"},
}
