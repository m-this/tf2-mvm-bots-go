BehaviorAction CTFBotMvMEngineerBuildTeleporter()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildTeleporter");

	action.OnStart = CTFBotMvMEngineerBuildTeleporter_OnStart;
	action.Update = CTFBotMvMEngineerBuildTeleporter_Update;
	action.OnEnd = CTFBotMvMEngineerBuildTeleporter_OnEnd;

	return action;
}

#define TELEPORTER_BUILD_MAX_TIME (40.0)

#define TELEPORTER_EXIT_REACH_TIME (12.0)

#define TELEPORTER_CLIMB_RISE_MIN (24.0)
#define TELEPORTER_CLIMB_RISE_MAX (72.0)
#define TELEPORTER_CLIMB_RANGE (140.0)
#define TELEPORTER_CLIMB_INTERVAL (0.7)
#define TELEPORTER_CLIMB_HOLD (0.3)
#define TELEPORTER_CLIMB_LIMIT (6)

#define TELEPORTER_BUILD_REACH (90.0)

#define TELEPORTER_SPAWN_OFFSET (200.0)
#define TELEPORTER_SPAWN_STEP (150.0)

#define TELEPORTER_EXIT_RADIUS (150.0)

#define TELEPORTER_EXIT_RADIUS_SAFE (500.0)

#define TELEPORTER_EXIT_RINGS (2)

#define TELEPORTER_TRY_POINTS (8)
#define TELEPORTER_TRY_TIME (1.5)

#define TELEPORTER_EXIT_TAKEN_RANGE (200.0)

float m_ctTeleporterGiveUp[65];
float m_ctTeleporterReachDeadline[65];
float m_ctTeleporterTryDeadline[65];
float m_ctTeleporterClimb[65];
int m_iTeleporterClimbs[65];
int m_iTeleporterTry[65];
TFObjectMode m_nTeleporterMode[65];
float m_vTeleporterSpot[65][3];
float m_vTeleporterStand[65][3];
float m_vTeleporterSpawn[65][3];
float m_vTeleporterNest[65][3];
float m_vTeleporterRouteSpot[65][8][3];
float m_vTeleporterRouteStand[65][8][3];
int m_iTeleporterRoutePoints[65];
bool m_bTeleporterNamedSpot[65];
bool m_bTeleporterGaveUp[65];
bool m_bTeleporterEntranceFirst[65];
bool m_bTeleporterEntranceFirstTried[65];
char m_sTeleporterLastResult[65][512];

// OnStart reads the route out of spawn while he is still standing at his nest.
public Action CTFBotMvMEngineerBuildTeleporter_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_ctTeleporterGiveUp[actor] = GetGameTime() + TELEPORTER_BUILD_MAX_TIME;
	m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
	m_ctTeleporterTryDeadline[actor] = GetGameTime() + TELEPORTER_TRY_TIME;
	m_ctTeleporterClimb[actor] = 0.0;
	m_iTeleporterClimbs[actor] = 0;
	m_iTeleporterTry[actor] = 0;
	m_iTeleporterRoutePoints[actor] = 0;
	// From the nest towards spawn when he stands at the nest, from spawn towards the nest when he
	// is still in it: the same route, read from whichever end he is at
	if ((m_nTeleporterMode[actor] == TFObjectMode_Entrance) && !m_bTeleporterNamedSpot[actor])
	{
		if (m_bTeleporterEntranceFirst[actor])
		{
			m_iTeleporterRoutePoints[actor] = SpawnRouteOut(actor, m_vTeleporterNest[actor], TELEPORTER_SPAWN_OFFSET, TELEPORTER_SPAWN_STEP, TELEPORTER_BUILD_REACH, m_vTeleporterRouteSpot[actor], m_vTeleporterRouteStand[actor], TELEPORTER_TRY_POINTS);
		}
		else
		{
			m_iTeleporterRoutePoints[actor] = SpawnRoutePoints(actor, m_vTeleporterSpawn[actor], TELEPORTER_SPAWN_OFFSET, TELEPORTER_SPAWN_STEP, TELEPORTER_BUILD_REACH, m_vTeleporterRouteSpot[actor], m_vTeleporterRouteStand[actor], TELEPORTER_TRY_POINTS);
		}
	}
	if (!TeleporterStandPoint(actor))
	{
		TeleporterGiveUp(actor);
		return TeleporterDone(action, actor, "No route out of spawn to walk");
	}
	// The half he is about to build is claimed, and the walk to it is a jump
	//
	// This is the walk the whole complaint was about: the entrance is at the far end of the map from
	// the nest, so building the pair costs the length of the map twice. The jump refuses itself
	// during a wave, so what a wave sees is the walk.
	if (Feature(FEATURE_ENGINEER_SETUP_PHASE))
	{
		ClaimSetupSpot(actor, TeleporterClaim(actor), m_vTeleporterSpot[actor]);
		SetupJump(actor, m_vTeleporterStand[actor]);
	}
	UpdateLookAroundForEnemies(actor, true);
	return action.Continue();
}

// Update walks to the spot, climbs onto it when the map put it on a rock, and
// presses fire.
public Action CTFBotMvMEngineerBuildTeleporter_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	// The sentry outranks this, always
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return TeleporterDone(action, actor, "Wave started");
	}
	if (!m_bTeleporterEntranceFirst[actor] && (GetObjectOfType(actor, TFObject_Sentry) == INVALID_ENT_REFERENCE))
	{
		return TeleporterDone(action, actor, "No sentry to leave behind");
	}
	if (GetObjectOfType(actor, TFObject_Teleporter, m_nTeleporterMode[actor]) != INVALID_ENT_REFERENCE)
	{
		g_arrPluginBot[actor].bPathing = false;
		return TeleporterDone(action, actor, "Built one");
	}
	if (m_ctTeleporterGiveUp[actor] < GetGameTime())
	{
		TeleporterGiveUp(actor);
		return TeleporterDone(action, actor, "Ran out of time");
	}
	// The walk to the named exit spot ran out, so he takes the ring round his own nest
	//
	// Where he stands when a walk fails is halfway to wherever he was going, and for the exit that is
	// the lane the robots come down. The nest ring is a spot rather than an accident: beside his own
	// sentry, out of the buster's blast, and it is where the exit goes on every map that names none.
	if (Feature(FEATURE_ENGINEER_CLIMBS) && (m_nTeleporterMode[actor] == TFObjectMode_Exit) && (GetGameTime() > m_ctTeleporterReachDeadline[actor]))
	{
		TeleporterFallBackToNest(actor);
	}
	float spot[3];
	spot = m_vTeleporterSpot[actor];
	// The walk to the exit ran out and the nest ring is gone too, so it goes down where he stands
	bool outOfTime = (m_nTeleporterMode[actor] == TFObjectMode_Exit) && (GetGameTime() > m_ctTeleporterReachDeadline[actor]);
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	// Say when the climb is not even asked for, so silence means one thing
	//
	// Three candidates for why the jump never lands, and the third is that this branch never runs.
	// Without a line here that reads the same as the debug being off.
	if (redbots_manager_debug_actions.BoolValue && Feature(FEATURE_ENGINEER_CLIMBS) && (outOfTime || !m_bTeleporterNamedSpot[actor]))
	{
		PrintToServer("[teleclimb] %N not asked: out of time %d, named spot %d", actor, outOfTime, m_bTeleporterNamedSpot[actor]);
	}
	// The map put the spot on top of something, so he gets on top of it rather than building below it
	if (Feature(FEATURE_ENGINEER_CLIMBS) && !outOfTime && m_bTeleporterNamedSpot[actor] && TeleporterClimbToSpot(actor, myBody, spot))
	{
		g_arrPluginBot[actor].bPathing = false;
		return action.Continue();
	}
	// Read after the climb, which moves it to where he landed
	float stand[3];
	stand = m_vTeleporterStand[actor];
	if (outOfTime)
	{
		stand = GetAbsOrigin(actor);
	}
	float teleporterRange = GetVectorDistance(GetAbsOrigin(actor), stand);
	// The toolbox comes out on the way in, so arriving is not another two seconds of standing about
	if (teleporterRange < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Teleporter, m_nTeleporterMode[actor]))
		{
			FakeClientCommandThrottled(actor, (m_nTeleporterMode[actor] == TFObjectMode_Entrance ? "build 1 0" : "build 1 1"));
		}
		// It goes where he looks, so he looks at the spot
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing teleporter");
	}
	if (teleporterRange > 70.0)
	{
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
		m_ctTeleporterTryDeadline[actor] = GetGameTime() + TELEPORTER_TRY_TIME;
		g_arrPluginBot[actor].SetPathGoalVector(stand);
		g_arrPluginBot[actor].bPathing = true;
		return action.Continue();
	}
	g_arrPluginBot[actor].bPathing = false;
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	if ((myWeapon != -1) && (TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER))
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");
		// The toolbox is out but the game has not decided yet
		if (objBeingBuilt == -1)
		{
			return action.Continue();
		}
		// This floor will not take it, so try the next place that might
		//
		// Only once he is actually looking at the spot: the answer while his head is still coming
		// round is the answer for wherever it was pointing, which is not this spot.
		if (!IsPlacementOK(objBeingBuilt) && !outOfTime && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctTeleporterTryDeadline[actor]))
		{
			m_iTeleporterTry[actor]++;
			if ((m_iTeleporterTry[actor] >= TeleporterTryLimit(actor)) || !TeleporterStandPoint(actor))
			{
				// The exit goes down here, and an entrance nowhere near the spawn door goes nowhere
				if (m_nTeleporterMode[actor] != TFObjectMode_Exit)
				{
					TeleporterGiveUp(actor);
					return TeleporterDone(action, actor, "Nowhere out of spawn takes one");
				}
				// Nothing round the named spot takes one, so the nest ring gets its own eight tries
				if (!Feature(FEATURE_ENGINEER_CLIMBS) || !TeleporterFallBackToNest(actor))
				{
					m_ctTeleporterReachDeadline[actor] = GetGameTime();
				}
				return action.Continue();
			}
			m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
			return action.Continue();
		}
	}
	VS_PressFireButton(actor);
	return action.Continue();
}

// TryLimit is how many placements he will try before he gives up.
//
// The exit walks two rings rather than one, so it gets two rounds of the same eight
// angles. Every other case has one spot or one route and is unchanged.
stock int TeleporterTryLimit(int actor)
{
	if (!m_bTeleporterNamedSpot[actor] && (m_nTeleporterMode[actor] == TFObjectMode_Exit))
	{
		return 16;
	}
	return TELEPORTER_TRY_POINTS;
}

// SayClimb says why a climb was refused, or that it was tried.
//
// Measured on Bigrock the jump never landed: no sample under 100 units from the spot,
// and the minimum equal to the median. Three candidates were left and one line
// separates them, because each writes a different reason here: the 24 to 72 window not
// matching the real rise, the jump not carrying, or the branch never being reached at
// all. See mvm-fgs.
stock void SayClimb(int actor, const char[] why, float rise, float flat)
{
	if (!redbots_manager_debug_actions.BoolValue)
	{
		return;
	}
	PrintToServer("[teleclimb] %N %s, rise %.0f of %.0f to %.0f, out %.0f of %.0f, climb %d of %d", actor, why, rise, TELEPORTER_CLIMB_RISE_MIN, TELEPORTER_CLIMB_RISE_MAX, flat, TELEPORTER_CLIMB_RANGE, m_iTeleporterClimbs[actor], TELEPORTER_CLIMB_LIMIT);
}

// ClimbToSpot crouch jumps onto the ground the spot sits on, and is false when there
// is nothing to climb.
//
// The stand point comes off the nav mesh, so for a spot on a rock the mesh does not
// cover it is the floor underneath: he arrives, the spot is over his head, and every
// placement from down there is refused. This puts him on top instead.
//
// Once he is up, where he stands is where he stands. Recomputing the ring point from up
// there asks the nav mesh again and the nav mesh answers with the floor he just left,
// which is the walk back down. He climbed from within a build's reach, so the spot is
// already in front of him.
//
// The count resets when he makes it, so falling off and climbing again costs another
// six attempts rather than none. The reach clock is what bounds the pair of them.
stock bool TeleporterClimbToSpot(int actor, IBody myBody, float spot[3])
{
	float origin[3];
	origin = GetAbsOrigin(actor);
	float rise = spot[2] - origin[2];
	float reach[3];
	SubtractVectors(spot, origin, reach);
	reach[2] = 0.0;
	float out = GetVectorLength(reach);
	if (rise < TELEPORTER_CLIMB_RISE_MIN)
	{
		SayClimb(actor, "nothing to climb", rise, out);
		if (m_iTeleporterClimbs[actor] > 0)
		{
			m_iTeleporterClimbs[actor] = 0;
			m_vTeleporterStand[actor] = origin;
		}
		return false;
	}
	// Higher than a crouch jump is not a ledge, it is a wall, and no number of jumps will do it
	if ((rise > TELEPORTER_CLIMB_RISE_MAX) || (m_iTeleporterClimbs[actor] >= TELEPORTER_CLIMB_LIMIT))
	{
		SayClimb(actor, (rise > TELEPORTER_CLIMB_RISE_MAX ? "too high to climb" : "out of climbs"), rise, out);
		return false;
	}
	// Far enough out and the jump lands on the wall rather than on top of it
	if (out > TELEPORTER_CLIMB_RANGE)
	{
		SayClimb(actor, "too far out to climb", rise, out);
		return false;
	}
	SayClimb(actor, "climbing", rise, out);
	AimHeadTowards(myBody, spot, MANDATORY, 0.2, Address_Null, "Climbing to the teleporter spot");
	if (m_ctTeleporterClimb[actor] > GetGameTime())
	{
		return true;
	}
	m_ctTeleporterClimb[actor] = GetGameTime() + TELEPORTER_CLIMB_INTERVAL;
	m_iTeleporterClimbs[actor]++;
	// Forward is along where he is looking, which is the spot, so the three together are a person
	g_arrExtraButtons[actor].PressButtons(IN_FORWARD | IN_JUMP | IN_DUCK, TELEPORTER_CLIMB_HOLD);
	return true;
}

// FallBackToNest is the named exit spot having beaten him, so he takes the ring round
// his own nest instead.
//
// Once per action: the named flag is what selects it and this clears it, so there is
// one fall back and then the ordinary give-up. False when there was no named spot to
// fall back from.
stock bool TeleporterFallBackToNest(int actor)
{
	if (!m_bTeleporterNamedSpot[actor] || (m_nTeleporterMode[actor] != TFObjectMode_Exit))
	{
		return false;
	}
	m_bTeleporterNamedSpot[actor] = false;
	m_iTeleporterTry[actor] = 0;
	m_iTeleporterClimbs[actor] = 0;
	m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
	return TeleporterStandPoint(actor);
}

// StandPoint is where this attempt puts the building, and where he stands to put it
// there.
//
// Three shapes, because the three cases are not the same question. A spot the map
// named is one spot and the man walks round it. The way out of spawn is a route rather
// than a spot, so the attempts walk along it, reading the points sampled off it when
// the action started. The exit has no spot at all, only a nest, so the spot walks round
// the nest and the man stands between the two.
//
// False when this attempt has nowhere left to put anything, which is the caller's cue
// to stop.
stock bool TeleporterStandPoint(int actor)
{
	int attempt = m_iTeleporterTry[actor];
	if (m_bTeleporterNamedSpot[actor])
	{
		float stand[3];
		BuildStandPoint(m_vTeleporterSpot[actor], GetAbsOrigin(actor), attempt, TELEPORTER_TRY_POINTS, TELEPORTER_BUILD_REACH, stand);
		m_vTeleporterStand[actor] = stand;
		return true;
	}
	if (m_nTeleporterMode[actor] == TFObjectMode_Exit)
	{
		float nest[3];
		nest = m_vTeleporterNest[actor];
		// The safe ring first, the whole way round, then the tight one
		float radius = 150.0;
		if (attempt < TELEPORTER_TRY_POINTS)
		{
			radius = TELEPORTER_EXIT_RADIUS_SAFE;
		}
		int angle = attempt % TELEPORTER_TRY_POINTS;
		// Both on the same ray out of the nest, so he stands a build's reach short of the spot
		float spot[3];
		BuildStandPoint(nest, GetAbsOrigin(actor), angle, TELEPORTER_TRY_POINTS, radius, spot);
		m_vTeleporterSpot[actor] = spot;
		float stand[3];
		BuildStandPoint(nest, GetAbsOrigin(actor), angle, TELEPORTER_TRY_POINTS, radius - TELEPORTER_BUILD_REACH, stand);
		m_vTeleporterStand[actor] = stand;
		return true;
	}
	// Past whatever another engineer has claimed, rather than onto it
	//
	// The route out of spawn is the same route for everybody who spawns there, so two engineers
	// reading it pick the same first point and stand in each other. The points are a hundred and
	// fifty apart and there are eight of them, so stepping past a claim costs a step.
	for (; attempt < m_iTeleporterRoutePoints[actor]; attempt++)
	{
		if (Feature(FEATURE_ENGINEER_SETUP_PHASE) && IsSetupSpotClaimed(actor, m_vTeleporterRouteSpot[actor][attempt]))
		{
			continue;
		}
		m_iTeleporterTry[actor] = attempt;
		m_vTeleporterSpot[actor] = m_vTeleporterRouteSpot[actor][attempt];
		m_vTeleporterStand[actor] = m_vTeleporterRouteStand[actor][attempt];
		return true;
	}
	return false;
}

// TeleporterClaim is which of the four setup spots this attempt is for.
stock int TeleporterClaim(int actor)
{
	if (m_nTeleporterMode[actor] == TFObjectMode_Exit)
	{
		return 3;
	}
	return 2;
}

// GiveUp stops the asking: for the break when the nest stood, for the early
// attempt alone when it did not, so the ordinary one still gets its turn.
stock void TeleporterGiveUp(int actor)
{
	if (m_bTeleporterEntranceFirst[actor])
	{
		m_bTeleporterEntranceFirstTried[actor] = true;
		return;
	}
	m_bTeleporterGaveUp[actor] = true;
}

// Ended is every way this action can end, so the reason survives it.
stock Action TeleporterDone(BehaviorAction action, int actor, const char[] reason)
{
	strcopy(m_sTeleporterLastResult[actor], 512, reason);
	return action.Done(reason);
}

// OnEnd stops the walking.
public void CTFBotMvMEngineerBuildTeleporter_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
}

// LastResult is why the last attempt ended, copied into the buffer sm_dump_nest
// handed in.
//
// //sp:name EngineerTeleporter_LastResult
// //sp:length buffer maxlength
stock void EngineerTeleporter_LastResult(int actor, char[] buffer, int maxlength)
{
	strcopy(buffer, maxlength, (m_sTeleporterLastResult[actor][0] == 0 ? "nothing yet" : m_sTeleporterLastResult[actor]));
}

// HasGivenUp says he stopped asking for this round.
stock bool EngineerTeleporter_HasGivenUp(int actor)
{
	return m_bTeleporterGaveUp[actor];
}

// Mode is the half he is building.
stock TFObjectMode EngineerTeleporter_Mode(int actor)
{
	return m_nTeleporterMode[actor];
}

// Spot is where this attempt puts it.
stock void EngineerTeleporter_Spot(int actor, float spot[3])
{
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	spot = m_vTeleporterSpot[actor];
	return;
}

// ResetBuildTeleporter forgets the early entrance for a seat, since the next bot
// in it is a different bot. Nothing else here is touched: the fields that were
// already carried between bots stay on the unreviewed list until mvm-z83.91
// decides them.
stock void Go_ResetBuildTeleporter(int client)
{
	m_bTeleporterEntranceFirst[client] = false;
	m_bTeleporterEntranceFirstTried[client] = false;
}

// ForgetGivingUp is a new wave being a new chance, and whatever refused him last
// time may have been a body standing on it.
stock void EngineerTeleporter_ForgetGivingUp()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_bTeleporterGaveUp[i] = false;
		m_bTeleporterEntranceFirst[i] = false;
		m_bTeleporterEntranceFirstTried[i] = false;
	}
}

// ShouldBuild is the half of the teleporter this engineer should go build, or none.
//
// Entrance before exit: an exit alone moves nobody, and the pair is only worth the
// metal once both ends stand. The entrance spot comes from the map configuration when
// it names one and from the way out of spawn when it does not, which is every official
// map; the exit goes beside the nest.
stock bool ShouldBuildTeleporter(int actor)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return false;
	}
	if (m_bTeleporterGaveUp[actor])
	{
		return false;
	}
	m_bTeleporterEntranceFirst[actor] = false;
	// The entrance first, while he still stands in spawn, when the switch says so: the nest is
	// picked by then and he is teleported onto it the moment the sentry action starts, so the
	// only walk this costs is the few hundred units out of the door
	if (Feature(FEATURE_ENGINEER_ENTRANCE_FIRST) && !m_bTeleporterEntranceFirstTried[actor] && (HasObjectOfType(actor, TFObject_Sentry, TFObjectMode_None) == INVALID_ENT_REFERENCE) && (GetObjectOfType(actor, TFObject_Teleporter, TFObjectMode_Entrance) == INVALID_ENT_REFERENCE))
	{
		return ShouldBuildEntranceFirst(actor);
	}
	// The nest comes first and it is not finished
	// What is in his hands counts: a carried building is one he has, not one he needs
	if (HasObjectOfType(actor, TFObject_Sentry, TFObjectMode_None) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (HasObjectOfType(actor, TFObject_Dispenser, TFObjectMode_None) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (m_aNestArea[actor] == NULL_AREA)
	{
		return false;
	}
	NestBuildPosition(m_aNestArea[actor], m_vTeleporterNest[actor]);
	if (GetObjectOfType(actor, TFObject_Teleporter, TFObjectMode_Entrance) == INVALID_ENT_REFERENCE)
	{
		m_nTeleporterMode[actor] = TFObjectMode_Entrance;
		float spot[3];
		bool named = NearestConfiguredSpot(g_arrMapConfig.adtTeleporterEntranceLocation, GetAbsOrigin(actor), spot);
		m_bTeleporterNamedSpot[actor] = named;
		m_vTeleporterSpot[actor] = spot;
		if (m_bTeleporterNamedSpot[actor])
		{
			return true;
		}
		// The map named none, which is most of them, so he walks out of spawn until the floor takes it
		float spawn[3];
		bool ok = NearestSpawnPoint(actor, spawn);
		m_vTeleporterSpawn[actor] = spawn;
		return ok;
	}
	if (GetObjectOfType(actor, TFObject_Teleporter, TFObjectMode_Exit) == INVALID_ENT_REFERENCE)
	{
		m_nTeleporterMode[actor] = TFObjectMode_Exit;
		// The nest itself when the map names no exit: the point of the pair is to arrive at the nest
		float spot[3];
		bool named = NearestFreeExitSpot(actor, m_vTeleporterNest[actor], spot);
		m_bTeleporterNamedSpot[actor] = named;
		m_vTeleporterSpot[actor] = spot;
		return true;
	}
	return false;
}

// ShouldBuildEntranceFirst is the early entrance: the nest must be picked, since
// the route out of spawn is read towards it, and the map's own spot wins when it
// names one.
stock bool ShouldBuildEntranceFirst(int actor)
{
	if (m_aNestArea[actor] == NULL_AREA)
	{
		return false;
	}
	NestBuildPosition(m_aNestArea[actor], m_vTeleporterNest[actor]);
	m_nTeleporterMode[actor] = TFObjectMode_Entrance;
	float spot[3];
	bool named = NearestConfiguredSpot(g_arrMapConfig.adtTeleporterEntranceLocation, GetAbsOrigin(actor), spot);
	m_bTeleporterNamedSpot[actor] = named;
	m_vTeleporterSpot[actor] = spot;
	if (m_bTeleporterNamedSpot[actor])
	{
		m_bTeleporterEntranceFirst[actor] = true;
		return true;
	}
	float spawn[3];
	bool ok = NearestSpawnPoint(actor, spawn);
	m_vTeleporterSpawn[actor] = spawn;
	m_bTeleporterEntranceFirst[actor] = ok;
	return ok;
}

// NearestFreeExitSpot is the nearest named exit spot another engineer has not already
// put one on.
//
// Coaltown names one exit and a team can field two engineers, and nothing stopped the
// second from building his on top of the first: reported from play as two exits sitting
// next to each other on the same platform. Two exits work, but the second one is a walk
// and fifty metal spent arriving where somebody could already arrive.
//
// With every named spot taken, this says no and the exit goes beside his own nest
// instead, which is where an exit is for anyway. The dispenser has had this rule for a
// while; the exit had not.
stock bool NearestFreeExitSpot(int actor, float nest[3], float spot[3])
{
	bool found;
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	ArrayList spots = g_arrMapConfig.adtTeleporterExitLocation;
	if (spots.Length == 0)
	{
		return false;
	}
	ArrayList free = new ArrayList(3);
	for (int i = 0; i < spots.Length; i++)
	{
		float candidate[3];
		spots.GetArray(i, candidate);
		if (Feature(FEATURE_ENGINEER_SETUP_PHASE) && IsSetupSpotClaimed(actor, candidate))
		{
			continue;
		}
		if (!IsExitSpotTaken(actor, candidate))
		{
			free.PushArray(candidate);
		}
	}
	found = NearestConfiguredSpot(free, nest, spot);
	delete free;
	return found;
}

// IsExitSpotTaken says somebody else's exit is already standing here.
stock bool IsExitSpotTaken(int actor, float spot[3])
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i))
		{
			continue;
		}
		int exitTele = GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Exit);
		if (exitTele == INVALID_ENT_REFERENCE)
		{
			continue;
		}
		if (GetVectorDistance(spot, GetAbsOrigin(exitTele)) < TELEPORTER_EXIT_TAKEN_RANGE)
		{
			return true;
		}
	}
	return false;
}

// NearestSpot is false when the map names no spot of this kind, which is most of
// them.
stock bool NearestConfiguredSpot(ArrayList spots, float from[3], float spot[3])
{
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	float nearest = -1.0;
	for (int i = 0; i < spots.Length; i++)
	{
		float candidate[3];
		spots.GetArray(i, candidate);
		float distance = GetVectorDistance(from, candidate);
		if ((nearest < 0.0) || (distance < nearest))
		{
			nearest = distance;
			spot = candidate;
		}
	}
	return nearest >= 0.0;
}

