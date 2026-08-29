/* Generated from internal/tables/wave.go. Do not edit.

The table it comes from is the only place these names are written. */

/* One line for the wave, with everything that was counted while it ran
 *
 * The duration is the honest number to compare runs on: a wave that is cleared slowly is a team
 * that nearly lost it, and a change that clears the same waves faster is a change that worked
 *
 * Not static, because spcomp scopes static to the file and this one is included: the helpers it
 * calls live in the plugin and the callers of it do too. */
void WriteWaveResult(const char[] result)
{
	/* A wave nobody played is not a result
	 *
	 * The game ends a wave when the round resets, which it does when the server restarts, so a
	 * restart wrote a row of zeros into the file. run.sh counts rows, so that row was the run: it
	 * stopped twenty seconds in and reported a wave lost that never began. Only a wave with a
	 * beginning is written.
	 */
	if (g_flWaveStart <= 0.0)
	{
		return;
	}

	CollectScoreboardHealing();

	float duration = GetGameTime() - g_flWaveStart;

	char line[STATS_LINE_LENGTH];
	FormatEx(line, sizeof(line),
		"{\"event\":\"wave_end\",\"map\":\"%s\",\"wave\":%d,\"result\":\"%s\",\"duration\":%.1f,"
		... "\"robot_kills\":%d,\"giant_kills\":%d,\"tank_kills\":%d,\"sentry_kills\":%d,\"defender_deaths\":%d,"
		... "\"backstabs\":%d,\"buster_detonations\":%d,\"sentries_lost\":%d,\"dispensers_lost\":%d,"
		... "\"upgrades\":%d,\"upgrade_credits\":%d,\"credits_dropped\":%d,\"credits_picked_up\":%d,"
		... "\"credits_bonus\":%d,\"credits_spent\":%d,\"credits_in_hand\":%d,\"damage\":%d,\"tank_damage\":%d,"
		... "\"sentry_damage\":%d,\"healing\":%d,\"ubers\":%d,\"damage_scout\":%d,\"damage_sniper\":%d,"
		... "\"damage_soldier\":%d,\"damage_demoman\":%d,\"damage_medic\":%d,\"damage_heavy\":%d,"
		... "\"damage_pyro\":%d,\"damage_spy\":%d,\"damage_engineer\":%d,\"kills_scout\":%d,"
		... "\"kills_soldier\":%d,\"kills_pyro\":%d,\"kills_demoman\":%d,\"kills_heavy\":%d,"
		... "\"kills_engineer\":%d,\"kills_medic\":%d,\"kills_sniper\":%d,\"kills_spy\":%d,"
		... "\"giantkills_scout\":%d,\"giantkills_soldier\":%d,\"giantkills_pyro\":%d,\"giantkills_demoman\":%d,"
		... "\"giantkills_heavy\":%d,\"giantkills_engineer\":%d,\"giantkills_medic\":%d,"
		... "\"giantkills_sniper\":%d,\"giantkills_spy\":%d,\"killedby_scout\":%d,\"killedby_soldier\":%d,"
		... "\"killedby_pyro\":%d,\"killedby_demoman\":%d,\"killedby_heavy\":%d,\"killedby_engineer\":%d,"
		... "\"killedby_medic\":%d,\"killedby_sniper\":%d,\"killedby_spy\":%d,\"killedby_sentry\":%d,"
		... "\"killedby_tank\":%d,\"cause_bullet\":%d,\"cause_explosion\":%d,\"cause_fire\":%d,"
		... "\"cause_melee\":%d,\"cause_backstab\":%d,\"cause_headshot\":%d,\"cause_fall\":%d,"
		... "\"cause_other\":%d,\"selfdamage_scout\":%d,\"selfdamage_soldier\":%d,\"selfdamage_pyro\":%d,"
		... "\"selfdamage_demoman\":%d,\"selfdamage_heavy\":%d,\"selfdamage_engineer\":%d,"
		... "\"selfdamage_medic\":%d,\"selfdamage_sniper\":%d,\"selfdamage_spy\":%d,\"selfdeaths_scout\":%d,"
		... "\"selfdeaths_soldier\":%d,\"selfdeaths_pyro\":%d,\"selfdeaths_demoman\":%d,\"selfdeaths_heavy\":%d,"
		... "\"selfdeaths_engineer\":%d,\"selfdeaths_medic\":%d,\"selfdeaths_sniper\":%d,\"selfdeaths_spy\":%d,"
		... "\"demo_pipe_damage\":%d,\"demo_sticky_damage\":%d,\"demo_melee_damage\":%d,"
		... "\"soldier_rocket_damage\":%d,\"soldier_other_damage\":%d,\"fired_soldier\":%d,\"hit_soldier\":%d,"
		... "\"fired_demoman\":%d,\"hit_demoman\":%d,\"jars_thrown\":%d,\"building_repaired\":%d,"
		... "\"building_damage\":%d,\"healing_scoreboard\":%d,\"healing_scout\":%d,\"healing_sniper\":%d,"
		... "\"healing_soldier\":%d,\"healing_demoman\":%d,\"healing_medic\":%d,\"healing_heavy\":%d,"
		... "\"healing_pyro\":%d,\"healing_spy\":%d,\"healing_engineer\":%d}",
		g_sMap,
		g_iWave,
		result,
		duration,
		g_Wave.robotKills,
		g_Wave.giantKills,
		g_Wave.tankKills,
		g_Wave.sentryKills,
		g_Wave.defenderDeaths,
		g_Wave.backstabs,
		g_Wave.busterDetonations,
		g_Wave.sentriesLost,
		g_Wave.dispensersLost,
		g_Wave.upgradesBought,
		g_Wave.upgradeCreditsSpent,
		g_Wave.creditsDropped,
		g_Wave.creditsAcquired,
		g_Wave.creditsBonus,
		g_Wave.creditsSpent,
		g_Wave.creditsInHand,
		g_Wave.damageDealt,
		g_Wave.damageToTanks,
		g_Wave.sentryDamage,
		g_Wave.healingDone,
		g_Wave.ubersDeployed,
		g_Wave.damageByClass[view_as<int>(TFClass_Scout)],
		g_Wave.damageByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.damageByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.damageByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.damageByClass[view_as<int>(TFClass_Medic)],
		g_Wave.damageByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.damageByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.damageByClass[view_as<int>(TFClass_Spy)],
		g_Wave.damageByClass[view_as<int>(TFClass_Engineer)],
		g_Wave.killsByClass[view_as<int>(TFClass_Scout)],
		g_Wave.killsByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.killsByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.killsByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.killsByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.killsByClass[view_as<int>(TFClass_Engineer)],
		g_Wave.killsByClass[view_as<int>(TFClass_Medic)],
		g_Wave.killsByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.killsByClass[view_as<int>(TFClass_Spy)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Scout)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Engineer)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Medic)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.giantKillsByClass[view_as<int>(TFClass_Spy)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Scout)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Soldier)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Pyro)],
		g_Wave.deathsToClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Heavy)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Engineer)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Medic)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Sniper)],
		g_Wave.deathsToClass[view_as<int>(TFClass_Spy)],
		g_Wave.deathsToSentry,
		g_Wave.deathsToTank,
		g_Wave.deathsByCause[DEATH_CAUSE_BULLET],
		g_Wave.deathsByCause[DEATH_CAUSE_EXPLOSION],
		g_Wave.deathsByCause[DEATH_CAUSE_FIRE],
		g_Wave.deathsByCause[DEATH_CAUSE_MELEE],
		g_Wave.deathsByCause[DEATH_CAUSE_BACKSTAB],
		g_Wave.deathsByCause[DEATH_CAUSE_HEADSHOT],
		g_Wave.deathsByCause[DEATH_CAUSE_FALL],
		g_Wave.deathsByCause[DEATH_CAUSE_OTHER],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Scout)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Engineer)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Medic)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.selfDamageByClass[view_as<int>(TFClass_Spy)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Scout)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Engineer)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Medic)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.selfDeathsByClass[view_as<int>(TFClass_Spy)],
		g_Wave.demoPipeDamage,
		g_Wave.demoStickyDamage,
		g_Wave.demoMeleeDamage,
		g_Wave.soldierRocketDamage,
		g_Wave.soldierOtherDamage,
		g_Wave.projectilesFired[view_as<int>(TFClass_Soldier)],
		g_Wave.projectilesHit[view_as<int>(TFClass_Soldier)],
		g_Wave.projectilesFired[view_as<int>(TFClass_DemoMan)],
		g_Wave.projectilesHit[view_as<int>(TFClass_DemoMan)],
		g_Wave.jarsThrown,
		g_Wave.buildingRepaired,
		g_Wave.buildingDamageTaken,
		g_Wave.healingScoreboard,
		g_Wave.healingByClass[view_as<int>(TFClass_Scout)],
		g_Wave.healingByClass[view_as<int>(TFClass_Sniper)],
		g_Wave.healingByClass[view_as<int>(TFClass_Soldier)],
		g_Wave.healingByClass[view_as<int>(TFClass_DemoMan)],
		g_Wave.healingByClass[view_as<int>(TFClass_Medic)],
		g_Wave.healingByClass[view_as<int>(TFClass_Heavy)],
		g_Wave.healingByClass[view_as<int>(TFClass_Pyro)],
		g_Wave.healingByClass[view_as<int>(TFClass_Spy)],
		g_Wave.healingByClass[view_as<int>(TFClass_Engineer)]);

	WriteLine(line);

	/* What the server's frames cost while that was happening

	Its own line, because it is about the machine rather than about the bots, and it should be
	possible to read a run's frame times without parsing everything else. */
	char perf[ENGINEER_LINE_LENGTH];
	FormatEx(perf, sizeof(perf),
		"{\"event\":\"perf\",\"map\":\"%s\",\"wave\":%d,\"frames\":%d,"
		... "\"frames_slow\":%d,\"frames_stalled\":%d,\"frame_mean_ms\":%.2f,\"frame_worst_ms\":%.1f,"
		... "\"red\":%d}",
		g_sMap, g_iWave, g_Wave.frames, g_Wave.framesSlow, g_Wave.framesStalled,
		g_Wave.frames > 0 ? g_Wave.frameTotalMs / float(g_Wave.frames) : 0.0,
		g_Wave.frameWorstMs, CountTeam(TFTeam_Red, false));

	WriteLine(perf);

	WriteEngineers("end");

	g_flWaveStart = 0.0;
}
