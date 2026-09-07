BehaviorAction CTFBotMvMEngineerBuildDispenser()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildDispenser");

	action.OnStart = CTFBotMvMEngineerBuildDispenser_OnStart;
	action.Update = CTFBotMvMEngineerBuildDispenser_Update;
	action.OnEnd = CTFBotMvMEngineerBuildDispenser_OnEnd;

	return action;
}

#define DISPENSER_SPOT_TAKEN_RANGE (150.0)

#define DISPENSER_SETTLE_RANGE (200.0)

#define DISPENSER_BUILD_REACH (90.0)
#define DISPENSER_TRY_POINTS (8)
#define DISPENSER_TRY_TIME (2.0)

#define DISPENSER_PRESS_SETTLE (0.3)

#define DISPENSER_BUILD_TIME (45.0)

float m_ctDispenserReachDeadline[65];
float m_ctDispenserGiveUpTime[65];
float m_ctDispenserTryDeadline[65];
float m_ctDispenserPressed[65];
int m_iDispenserTry[65];
float m_vDispenserSpot[65][3];
float m_vDispenserStand[65][3];

// OnStart picks the spot once and prices the walk to it.
public Action CTFBotMvMEngineerBuildDispenser_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
	m_ctDispenserGiveUpTime[actor] = GetGameTime() + DISPENSER_BUILD_TIME;
	m_ctDispenserTryDeadline[actor] = GetGameTime() + DISPENSER_TRY_TIME;
	m_ctDispenserPressed[actor] = 0.0;
	m_iDispenserTry[actor] = 0;
	// Once, here, because the Update runs every tick and a path computation does not belong there
	float spot[3];
	bool configured = ConfiguredDispenserSpot(actor, spot);
	m_vDispenserSpot[actor] = spot;
	if (!configured)
	{
		if (m_aNestArea[actor] != NULL_AREA)
		{
			CNavArea_GetRandomPoint(m_aNestArea[actor], m_vDispenserSpot[actor]);
		}
		else
		{
			m_vDispenserSpot[actor] = GetAbsOrigin(actor);
		}
	}
	ClimbBegin(actor);
	// Sides he cannot stand on are skipped here rather than walked at and waited out
	float stand[3];
	bool ok = DispenserStandPoint(actor, m_iDispenserTry[actor], stand);
	m_vDispenserStand[actor] = stand;
	if (!ok)
	{
		NextDispenserStandPoint(actor);
	}
	// Level with a spot on a rock already, so the stand point is not the floor below it
	if (ClimbBeside(actor, m_vDispenserSpot[actor]))
	{
		ClimbMarkOnTop(actor, m_vDispenserStand[actor]);
		m_vDispenserStand[actor] = GetAbsOrigin(actor);
	}
	// Claimed and jumped to, so the walk from the upgrade station is not paid twice
	if (Feature(FEATURE_ENGINEER_SETUP_PHASE))
	{
		ClaimSetupSpot(actor, 1, m_vDispenserSpot[actor]);
		SetupJump(actor, m_vDispenserStand[actor]);
	}
	// Priced by the walk, because the spot the map names is not always next to the nest
	//
	// Coaltown's right-hand spot is 857 units from the nest it serves, on purpose, and he starts the
	// walk at the upgrade station. A flat twelve seconds expired somewhere along the way and he built
	// it wherever that was, which is how a hand-walked spot turned into a dispenser beside the
	// teleporter entrance.
	m_ctDispenserReachDeadline[actor] = GetGameTime() + BuildReachTime(GetAbsOrigin(actor), m_vDispenserStand[actor]);
	return action.Continue();
}

// Update walks to the stand point and presses once per settle until one goes
// down.
public Action CTFBotMvMEngineerBuildDispenser_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_aNestArea[actor] == NULL_AREA)
	{
		LogBuildFailure(actor, "dispenser", "no nest area");
		return action.Done("No hint entity");
	}
	int sentry = GetObjectOfType(actor, TFObject_Sentry);
	if (sentry == INVALID_ENT_REFERENCE)
	{
		LogBuildFailure(actor, "dispenser", "no sentry to feed");
		return action.Done("No sentry");
	}
	// Asked of the sentry, not of the flag the idle action keeps
	//
	// Suspending the idle action stops its update running, so its three second flag expires three
	// seconds after this one starts however well the sentry is doing. This ended itself on that,
	// every time, and only ever finished a dispenser where the walk and the placement both fitted
	// inside those three seconds.
	if (!IsSentrySafe(sentry))
	{
		LogBuildFailure(actor, "dispenser", "sentry under fire");
		return action.Done("Sentry not safe");
	}
	if (CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(actor))
	{
		LogBuildFailure(actor, "dispenser", "told to advance the nest");
		return action.Done("Need to advance nest");
	}
	// The spot is chosen once, not every frame
	//
	// 	Choosing it here used to mean a path computation per configured spot per tick per engineer,
	// 	which is how the server's watchdog came to fire inside NavAreaBuildPath. A spot that was
	// 	reachable when the action started is reachable a second later, and if it is not, the deadline
	// 	below is what answers for it.
	// Every side of the spot refused him, and a wave held for one dispenser is the worse trade
	if (GetGameTime() > m_ctDispenserGiveUpTime[actor])
	{
		LogBuildFailure(actor, "dispenser", "ran out of time to place it");
		return action.Done("Nowhere to put a dispenser");
	}
	float spot[3];
	spot = m_vDispenserSpot[actor];
	float stand[3];
	stand = m_vDispenserStand[actor];
	IBody myBody = CBaseNPC_GetNextBotOfEntity(actor).GetBodyInterface();
	// The spot is on a rock and he is at the foot of it, so he gets on top before the clocks are read
	int climbed = ClimbToSpot(actor, myBody, spot, "dispenser spot");
	if (climbed == 1)
	{
		g_arrPluginBot[actor].bPathing = false;
		return action.Continue();
	}
	if (climbed == 2)
	{
		m_ctDispenserReachDeadline[actor] = GetGameTime() + BuildReachTime(spot, spot);
	}
	// Up on the rock nothing is pathed to: where he stands is the stand point, stepped off the spot
	if (ClimbOnTop(actor) && ClimbBeside(actor, spot))
	{
		m_vDispenserStand[actor] = GetAbsOrigin(actor);
		stand = m_vDispenserStand[actor];
		if (ClimbStepBack(actor, myBody, spot))
		{
			g_arrPluginBot[actor].bPathing = false;
			return action.Continue();
		}
	}
	// The walk ran out of time, so he builds from where he stands and aims at the spot anyway
	//
	// Only while he is somewhere near his nest. Settling where he stands is a trade of accuracy for a
	// dispenser that exists, and it stops being a trade at all once he is far enough away that what
	// he settles for feeds nothing.
	bool outOfTime = (m_ctDispenserReachDeadline[actor] > 0.0) && (GetGameTime() > m_ctDispenserReachDeadline[actor]) && (GetVectorDistance(GetAbsOrigin(actor), spot) < DISPENSER_SETTLE_RANGE);
	if (outOfTime)
	{
		stand = GetAbsOrigin(actor);
	}
	// He never arrived, so the spot is unreachable rather than slow
	//
	// outOfTime above settles for where he stands, and only while he is near the nest, for the reason
	// in the comment on it. That leaves an engineer who never gets near at all walking at the same
	// spot for the rest of the mission.
	//
	// He gives the dispenser up rather than standing there. The action ends, the idle behaviour picks
	// again, and a dispenser he does not have is worth less than an engineer who is doing something
	// else.
	if ((m_ctDispenserReachDeadline[actor] > 0.0) && (GetGameTime() > m_ctDispenserReachDeadline[actor]) && (GetVectorDistance(GetAbsOrigin(actor), spot) >= DISPENSER_SETTLE_RANGE))
	{
		LogBuildFailure(actor, "dispenser", "could not reach the spot, gave it up");
		return action.Done("Cannot reach the dispenser spot");
	}
	float rangeToStand = GetVectorDistance(GetAbsOrigin(actor), stand);
	if (rangeToStand < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Dispenser))
		{
			FakeClientCommandThrottled(actor, "build 0");
		}
		// It goes where he looks, so he looks at the spot. Turning on the spot only turns the problem
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing dispenser");
	}
	// NOTE: we do not look around for incoming enemies cause all we care about is placing this dispenser
	if (rangeToStand > 70.0)
	{
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
		// The game says no from here, so try looking at it from the next side
		//
		// Only once he is actually looking at the spot: the answer while his head is still coming
		// round is the answer for wherever it was pointing, which is not this spot.
		if (!IsPlacementOK(objBeingBuilt) && !outOfTime && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctDispenserTryDeadline[actor]))
		{
			NextDispenserStandPoint(actor);
			return action.Continue();
		}
	}
	// Asked before the press, not after it
	//
	// It used to press and then ask in the same frame, which is a frame too early: the answer was
	// always "no dispenser", so the next tick pressed again and put a second one down.
	int dispenser = GetObjectOfType(actor, TFObject_Dispenser);
	if (dispenser != INVALID_ENT_REFERENCE)
	{
		SayIfBuiltElsewhere(actor, dispenser, m_vDispenserSpot[actor], "dispenser");
		SetPlayerReady(actor, true);
		return action.Done("Built a dispenser");
	}
	// A press already given its chance is not given another until the game has had its tick
	if (GetGameTime() < m_ctDispenserPressed[actor])
	{
		return action.Continue();
	}
	m_ctDispenserPressed[actor] = GetGameTime() + DISPENSER_PRESS_SETTLE;
	VS_PressFireButton(actor);
	return action.Continue();
}

// StandPoint is one build's reach short of the spot, on the side the try asks for,
// and on ground he can stand on.
//
// Try zero is the side he is walking in from, so the first look costs him no walking
// at all. Each one after it is forty five degrees round from there. False when the nav
// mesh has nothing walkable there, which is the caller's cue to go round to the next
// one.
stock bool DispenserStandPoint(int actor, int attempt, float stand[3])
{
	bool ok;
	for (int i = 0; i < 3; i++)
	{
		stand[i] = 0.0;
	}
	// A side level with the spot before a side a storey under it, see LevelStandPoint
	int side;
	ok = LevelStandPoint(m_vDispenserSpot[actor], GetAbsOrigin(actor), attempt, DISPENSER_TRY_POINTS, DISPENSER_BUILD_REACH, side, stand);
	m_iDispenserTry[actor] = side;
	return ok;
}

// NextStandPoint is the next side of the spot he can actually stand on, or the
// end of them, which is when he settles.
stock void NextDispenserStandPoint(int actor)
{
	for (m_iDispenserTry[actor]++; m_iDispenserTry[actor] < DISPENSER_TRY_POINTS; m_iDispenserTry[actor]++)
	{
		float stand[3];
		bool ok = DispenserStandPoint(actor, m_iDispenserTry[actor], stand);
		m_vDispenserStand[actor] = stand;
		if (!ok)
		{
			continue;
		}
		m_ctDispenserTryDeadline[actor] = GetGameTime() + DISPENSER_TRY_TIME;
		return;
	}
	// A dispenser two metres from the spot beats an engineer who never builds one
	m_ctDispenserReachDeadline[actor] = GetGameTime();
}

// OnEnd puts his eyes back on the wave.
public void CTFBotMvMEngineerBuildDispenser_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
}

// ConfiguredSpot is the dispenser spot the map configuration asks for, false when it
// asks for nothing.
//
// Nearest to the nest rather than to the engineer, because he walks back to the nest
// anyway and the dispenser is there to feed the sentry.
stock bool ConfiguredDispenserSpot(int actor, float spot[3])
{
	bool found;
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	ArrayList spots = g_arrMapConfig.adtDispenserLocation;
	if (spots.Length == 0)
	{
		return false;
	}
	// The authored point rather than the area centre, so the comparison is like with like
	float nest[3];
	NestBuildPosition(m_aNestArea[actor], nest);
	// The zone this nest belongs to, when the map names one, decides before distance does
	//
	// Coaltown is why. The ground behind the wall on the right is eight hundred units from the nest
	// it serves and two hundred from a different one, so nearest is simply the wrong answer there and
	// no distance rule was ever going to fix it: the map has to be able to say which spot goes with
	// which nest, and a zone is how it already says that about nests.
	char myZone[512];
	NestZoneOf(m_aNestArea[actor], myZone, 512);
	// His own zone if the map put a spot in it, and the spots belonging to nobody if it did not
	//
	// Two passes rather than one condition, because the two ideas are different. A spot in a zone is
	// reserved for it: Coaltown's right building has one and nothing else may take it, or the nearest
	// rule hands it to the nest in the middle. A nest in a zone is a separate and older idea, about
	// spreading engineers over the map, and it must not stop that engineer using a spot nobody
	// reserved. Mannhattan names zones on all four of its nests and on none of its spots, and one
	// condition covering both left it with nothing at all.
	ArrayList free = new ArrayList(3);
	// A spot the path query refused is the last resort, not a spot that stopped existing
	//
	// The query is the same ComputeToPos that was measured returning nothing at all for a medic with
	// a live patient in front of him, and it is asked here from wherever the engineer happens to be
	// standing. Before the first wave that is his nest, and it answers; between later waves it is the
	// upgrade station at the other end of the map, and when it refuses, the coordinate somebody
	// walked the map to find is silently dropped and he builds wherever the fallback puts him.
	//
	// Reported exactly that way: right before wave one, wrong from then on. So an unreachable
	// authored spot is still an authored spot, and it is used when nothing better is offered.
	ArrayList refused = new ArrayList(3);
	CollectDispenserSpots(actor, myZone, free, refused);
	if ((free.Length == 0) && (myZone[0] != 0))
	{
		CollectDispenserSpots(actor, "", free, refused);
	}
	found = NearestConfiguredSpot(free, nest, spot);
	if (!found)
	{
		found = NearestConfiguredSpot(refused, nest, spot);
	}
	if (redbots_manager_debug.BoolValue)
	{
		if (found)
		{
			PrintToServer("ConfiguredDispenserSpot: %N takes the named spot %.0f %.0f %.0f", actor, spot[0], spot[1], spot[2]);
		}
		else
		{
			PrintToServer("ConfiguredDispenserSpot: %N has no named spot for the nest at %.0f %.0f %.0f", actor, nest[0], nest[1], nest[2]);
		}
	}
	delete refused;
	delete free;
	return found;
}

// CollectSpots is every named spot in one zone nobody has taken, split by whether
// the mesh will admit a path today.
stock void CollectDispenserSpots(int actor, const char[] wanted, ArrayList free, ArrayList refused)
{
	ArrayList spots = g_arrMapConfig.adtDispenserLocation;
	ArrayList zones = g_arrMapConfig.adtDispenserZone;
	for (int i = 0; i < spots.Length; i++)
	{
		char zone[512];
		if (i < zones.Length)
		{
			zones.GetString(i, zone, 512);
		}
		if (!StrEqual(zone, wanted))
		{
			continue;
		}
		float candidate[3];
		spots.GetArray(i, candidate);
		// Somebody else's dispenser standing on it is the one thing that does rule a spot out
		if (IsDispenserSpotTaken(actor, candidate))
		{
			continue;
		}
		if (IsPathToVectorPossible(actor, candidate))
		{
			free.PushArray(candidate);
		}
		else
		{
			refused.PushArray(candidate);
		}
	}
}

// IsSpotTaken spreads several engineers over the spots the map names instead of
// stacking them on the nearest one.
stock bool IsDispenserSpotTaken(int actor, float spot[3])
{
	// Claimed counts as taken: the point of claiming is to be read before anything stands there
	if (Feature(FEATURE_ENGINEER_SETUP_PHASE) && IsSetupSpotClaimed(actor, spot))
	{
		return true;
	}
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i))
		{
			continue;
		}
		int dispenser = GetObjectOfType(i, TFObject_Dispenser);
		if (dispenser == INVALID_ENT_REFERENCE)
		{
			continue;
		}
		if (GetVectorDistance(spot, GetAbsOrigin(dispenser)) < DISPENSER_SPOT_TAKEN_RANGE)
		{
			return true;
		}
	}
	return false;
}

