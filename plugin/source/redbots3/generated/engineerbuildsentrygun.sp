BehaviorAction CTFBotMvMEngineerBuildSentrygun()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildSentrygun");

	action.OnStart = CTFBotMvMEngineerBuildSentrygun_OnStart;
	action.Update = CTFBotMvMEngineerBuildSentrygun_Update;
	action.OnEnd = CTFBotMvMEngineerBuildSentrygun_OnEnd;

	return action;
}

#define SENTRY_BUILD_REACH (90.0)

#define SENTRY_TRY_POINTS (8)

#define SENTRY_PRESS_SETTLE (0.3)

#define SENTRY_TRY_TIME (1.5)

#define SENTRY_REACH_TIME (12.0)
#define SENTRY_SETTLE_RANGE (200.0)
#define SENTRY_BUILD_TIME (45.0)

#define SENTRY_STUCK_GIVEUP (2)

float m_ctSentryReachDeadline[65];
int m_iSentryStuckMark[65];
CNavArea m_aSentryStuckArea[65];
float m_ctSentryGiveUpTime[65];
float m_ctSentryTryDeadline[65];
float m_ctSentryPressed[65];
int m_iSentryTry[65];
float m_vSentrySpot[65][3];
float m_vSentryStand[65][3];

// OnStart arms every clock, teleports him onto the nest between rounds, and
// marks where the stuck count stood.
public Action CTFBotMvMEngineerBuildSentrygun_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
	m_ctSentryGiveUpTime[actor] = GetGameTime() + SENTRY_BUILD_TIME;
	m_ctSentryTryDeadline[actor] = GetGameTime() + SENTRY_TRY_TIME;
	m_ctSentryPressed[actor] = 0.0;
	m_iSentryTry[actor] = 0;
	if (GameRules_GetRoundState() == RoundState_BetweenRounds)
	{
		if (m_aNestArea[actor] != NULL_AREA)
		{
			// Teleport ourselves to the nest area for a faster setup
			float vNestPosition[3];
			NestBuildPosition(m_aNestArea[actor], vNestPosition);
			vNestPosition[2] += TFBOT_STEP_HEIGHT;
			CBaseEntity(actor).SetAbsOrigin(vNestPosition);
			// The nest is the first claim of the break, and the one the other three are placed around
			if (Feature(FEATURE_ENGINEER_SETUP_PHASE))
			{
				ClaimSetupSpot(actor, 0, vNestPosition);
			}
		}
	}
	ClimbBegin(actor);
	SentryStandPoint(actor);
	// The teleport put him level with a spot on a rock, so he builds from up here
	//
	// The stand point comes off the mesh and the mesh has nothing on the rock, so it is the floor
	// below, and walking to it is walking off the ground the break just put him on. Bigrock's two
	// nests are this.
	if (ClimbBeside(actor, m_vSentrySpot[actor]))
	{
		ClimbMarkOnTop(actor, m_vSentryStand[actor]);
		m_vSentryStand[actor] = GetAbsOrigin(actor);
	}
	// After the teleport above, so a between-rounds walk is priced from where he actually starts it
	m_ctSentryReachDeadline[actor] = GetGameTime() + BuildReachTime(GetAbsOrigin(actor), m_vSentryStand[actor]);
	LogBuildFailure(actor, "sentry", "started");
	// The mark has to survive the watchdog's reset, which is the whole point of it
	//
	// ResetIntentionInterface restarts this action, so anything armed in OnStart is armed again every
	// twelve seconds and can never expire. That is the fault in the reach deadline, and re-marking
	// here would inherit it: the count would restart alongside the thing it counts.
	//
	// So the mark belongs to the nest rather than to the attempt. It resets when he is sent somewhere
	// new, and not when he is merely restarted at the same place.
	if (m_aSentryStuckArea[actor] != m_aNestArea[actor])
	{
		m_aSentryStuckArea[actor] = m_aNestArea[actor];
		m_iSentryStuckMark[actor] = StuckCountOf(actor);
		LogBuildFailure(actor, "sentry", "new nest, stuck mark reset");
	}
	return action.Continue();
}

// Update walks to the stand point, gets the toolbox out, and presses once per
// settle until the game accepts one.
public Action CTFBotMvMEngineerBuildSentrygun_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_aNestArea[actor] == NULL_AREA)
	{
		LogBuildFailure(actor, "sentry", "no nest area");
		return action.Done("No hint entity");
	}
	if (CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(actor))
	{
		LogBuildFailure(actor, "sentry", "told to advance the nest");
		return action.Done("No sentry");
	}
	// Every side of this spot refused him and the walk is not getting shorter. The idle action retries
	if (GetGameTime() > m_ctSentryGiveUpTime[actor])
	{
		LogBuildFailure(actor, "sentry", "every side of the spot refused him");
		return action.Done("Nowhere here will take a sentry");
	}
	float spot[3];
	spot = m_vSentrySpot[actor];
	float stand[3];
	stand = m_vSentryStand[actor];
	IBody myBody = CBaseNPC_GetNextBotOfEntity(actor).GetBodyInterface();
	// The spot is on a rock and he is at the foot of it, so he gets on top before anything else
	//
	// Before the clocks below, because a jump in flight is not a walk that ran out. Where he lands, the
	// stand point is a short reach from the spot up here: asking the mesh answers with the floor he
	// just left, and standing on the spot itself is looking at his own feet.
	int climbed = ClimbToSpot(actor, myBody, spot, "sentry spot");
	if (climbed == 1)
	{
		g_arrPluginBot[actor].bPathing = false;
		return action.Continue();
	}
	if (climbed == 2)
	{
		m_ctSentryReachDeadline[actor] = GetGameTime() + SENTRY_REACH_TIME;
	}
	// Up on the rock, where he stands is the stand point and nothing is pathed to
	//
	// A point up here is off the mesh, and a path asked for to it is a search the watchdog ends. He is
	// steered a short reach off the spot instead, so that he is not looking at his own feet.
	if (ClimbOnTop(actor) && ClimbBeside(actor, spot))
	{
		m_vSentryStand[actor] = GetAbsOrigin(actor);
		stand = m_vSentryStand[actor];
		if (ClimbStepBack(actor, myBody, spot))
		{
			g_arrPluginBot[actor].bPathing = false;
			return action.Continue();
		}
	}
	// The walk ran out, so he builds from where he got to rather than into whatever stopped him
	//
	// And he puts it beside himself rather than pointing it at the nest he could not reach. Aiming at
	// the nest from three metres short of it is the same thing; aiming at it from twenty metres short
	// puts the sentry twenty metres from where anybody wanted it, facing a direction chosen by where
	// he happened to get stuck. Decoy produced one 625 units from its own nest that way.
	float rangeToSpot = GetVectorDistance(GetAbsOrigin(actor), m_vSentrySpot[actor]);
	bool outOfTime = (GetGameTime() > m_ctSentryReachDeadline[actor]) && (rangeToSpot < SENTRY_SETTLE_RANGE);
	// The walk ran out and he is nowhere near the spot, so the spot is what to give up on
	//
	// outOfTime above deliberately refuses to build from far away, for the reason in the comment on
	// it. What that leaves is the case nothing handled: an engineer who never arrives keeps walking at
	// a spot he cannot reach, for the whole mission, and builds nothing at all. Reported on Mannworks
	// with Mean Machines, and Bigrock has a spot on a rock he cannot jump onto.
	//
	// The retry below re-scores the nest, and it only runs once he is close enough to try building. So
	// the same thing is done here, from the other side of the range check: a new area rather than a
	// sentry twenty metres from where anybody wanted one.
	if (((GetGameTime() > m_ctSentryReachDeadline[actor]) && (rangeToSpot >= SENTRY_SETTLE_RANGE)) || ((StuckCountOf(actor) - m_iSentryStuckMark[actor]) >= SENTRY_STUCK_GIVEUP))
	{
		m_iSentryStuckMark[actor] = StuckCountOf(actor);
		m_aSentryStuckArea[actor] = m_aNestArea[actor];
		m_aNestArea[actor] = PickBuildArea(actor);
		m_iSentryTry[actor] = 0;
		ClimbBegin(actor);
		SentryStandPoint(actor);
		m_ctSentryReachDeadline[actor] = GetGameTime() + SENTRY_REACH_TIME;
		LogBuildFailure(actor, "sentry", "could not reach the spot, took another");
		return action.Continue();
	}
	if (outOfTime)
	{
		stand = GetAbsOrigin(actor);
		BuildStandPoint(stand, m_vSentrySpot[actor], m_iSentryTry[actor], SENTRY_TRY_POINTS, SENTRY_BUILD_REACH, spot);
	}
	float rangeToStand = GetVectorDistance(GetAbsOrigin(actor), stand);
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	ILocomotion myLoco = CBaseNPC_GetNextBotOfEntity(actor).GetLocomotionInterface();
	if (rangeToStand < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Sentry))
		{
			FakeClientCommandThrottled(actor, "build 2");
		}
		UpdateLookAroundForEnemies(actor, false);
		if (!myLoco.IsStuck())
		{
			g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
		}
		// It goes where he looks, so he looks at the spot rather than at the ground under himself
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing sentry");
	}
	if (rangeToStand > 70.0)
	{
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
		m_ctSentryTryDeadline[actor] = GetGameTime() + SENTRY_TRY_TIME;
		g_arrPluginBot[actor].SetPathGoalVector(stand);
		g_arrPluginBot[actor].bPathing = true;
		if (rangeToStand > 300.0)
		{
			EquipWeaponSlot(actor, TFWeaponSlot_Primary);
		}
		UpdateLookAroundForEnemies(actor, true);
		return action.Continue();
	}
	g_arrPluginBot[actor].bPathing = false;
	if ((myWeapon != -1) && (TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER))
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");
		if (objBeingBuilt == -1)
		{
			return action.Continue();
		}
		// One press, then a tick for the game to act on it
		//
		// The check at the end of this function runs in the same frame as this press, so it asks
		// whether a sentry exists before the game has put one down. It answered no, the action
		// carried on, and the toolbox re-armed: another press, another building. Measured on the
		// dispenser, which has the same shape and which the test-bed caught standing twice under one
		// engineer.
		if (GetGameTime() >= m_ctSentryPressed[actor])
		{
			m_ctSentryPressed[actor] = GetGameTime() + SENTRY_PRESS_SETTLE;
			VS_PressFireButton(actor);
		}
		// The game says no from here, so try looking at it from the next side round
		//
		// Only once he is actually looking at it: the answer while his head is still coming round is
		// the answer for wherever it was pointing, which is not this spot.
		if (!IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctSentryTryDeadline[actor]))
		{
			m_iSentryTry[actor]++;
			// Every side refused him, so now the spot itself is the thing in question
			//
			// This is where the nest gets re-scored, and not before: a pass over the nav mesh is the
			// expensive answer and it was being given to a wall behind the man.
			if (m_iSentryTry[actor] >= SENTRY_TRY_POINTS)
			{
				m_aNestArea[actor] = PickBuildArea(actor);
				m_iSentryTry[actor] = 0;
				ClimbBegin(actor);
			}
			SentryStandPoint(actor);
			// Still level with the spot, so the next side is looked at from up here rather than from below
			if (ClimbBeside(actor, m_vSentrySpot[actor]))
			{
				ClimbMarkOnTop(actor, m_vSentryStand[actor]);
				m_vSentryStand[actor] = GetAbsOrigin(actor);
			}
			m_ctSentryTryDeadline[actor] = GetGameTime() + SENTRY_TRY_TIME;
			m_ctSentryReachDeadline[actor] = GetGameTime() + SENTRY_REACH_TIME;
			return action.Continue();
		}
	}
	int sentry = GetObjectOfType(actor, TFObject_Sentry);
	if (sentry == INVALID_ENT_REFERENCE)
	{
		return action.Continue();
	}
	SayIfBuiltElsewhere(actor, sentry, m_vSentrySpot[actor], "sentry");
	SetPlayerReady(actor, true);
	LogBuildFailure(actor, "sentry", "built one");
	return action.Done("Built a sentry");
}

// StandPoint is where the sentry goes and where he stands to put it there, on a
// side he can stand on.
//
// Sides with nothing walkable under them are skipped rather than walked at: a nest
// on raised ground has thin air around it, and pathing at a coordinate in mid-air
// puts the engineer on the floor below holding the toolbox until a clock saves him.
// Bounded by the number of sides there are.
stock void SentryStandPoint(int actor)
{
	NestBuildPosition(m_aNestArea[actor], m_vSentrySpot[actor]);
	for (int skipped = 0; skipped < SENTRY_TRY_POINTS; skipped++)
	{
		float stand[3];
		bool ok = BuildStandPoint(m_vSentrySpot[actor], GetAbsOrigin(actor), m_iSentryTry[actor], SENTRY_TRY_POINTS, SENTRY_BUILD_REACH, stand);
		m_vSentryStand[actor] = stand;
		if (ok)
		{
			return;
		}
		m_iSentryTry[actor] = (m_iSentryTry[actor] + 1) % SENTRY_TRY_POINTS;
	}
}

// OnEnd stops the walking and says what the attempt left behind.
public void CTFBotMvMEngineerBuildSentrygun_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
	// Every way out of this action, including the ones nobody wrote a branch for
	//
	// The Done branches above name why they gave up, and a session produced far more starts than
	// endings that said anything. Asking the result for its reason here printed nothing at all, which
	// is what a thrown native looks like from the outside: it takes the callback with it. So this says
	// only what is certainly true, which is that the attempt is over and whether it left a sentry.
	LogBuildFailure(actor, "sentry", (GetObjectOfType(actor, TFObject_Sentry) != INVALID_ENT_REFERENCE ? "ended with a sentry" : "ended with nothing"));
}

