BehaviorAction CTFBotMvMEngineerBuildSentrygun()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildSentrygun");

	action.OnStart = CTFBotMvMEngineerBuildSentrygun_OnStart;
	action.Update = CTFBotMvMEngineerBuildSentrygun_Update;
	action.OnEnd = CTFBotMvMEngineerBuildSentrygun_OnEnd;

	return action;
}

#define Go_Slots (65)

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
			float vNestPosition[3];
			NestBuildPosition(m_aNestArea[actor], vNestPosition);
			vNestPosition[2] += TFBOT_STEP_HEIGHT;
			CBaseEntity(actor).SetAbsOrigin(vNestPosition);
		}
	}
	SentryStandPoint(actor);
	m_ctSentryReachDeadline[actor] = GetGameTime() + BuildReachTime(GetAbsOrigin(actor), m_vSentryStand[actor]);
	LogBuildFailure(actor, "sentry", "started");
	if (m_aSentryStuckArea[actor] != m_aNestArea[actor])
	{
		m_aSentryStuckArea[actor] = m_aNestArea[actor];
		m_iSentryStuckMark[actor] = StuckCountOf(actor);
		LogBuildFailure(actor, "sentry", "new nest, stuck mark reset");
	}
	return action.Continue();
}

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
	if (GetGameTime() > m_ctSentryGiveUpTime[actor])
	{
		LogBuildFailure(actor, "sentry", "every side of the spot refused him");
		return action.Done("Nowhere here will take a sentry");
	}
	float spot[3];
	spot = m_vSentrySpot[actor];
	float stand[3];
	stand = m_vSentryStand[actor];
	float rangeToSpot = GetVectorDistance(GetAbsOrigin(actor), m_vSentrySpot[actor]);
	bool outOfTime = (GetGameTime() > m_ctSentryReachDeadline[actor]) && (rangeToSpot < SENTRY_SETTLE_RANGE);
	if (((GetGameTime() > m_ctSentryReachDeadline[actor]) && (rangeToSpot >= SENTRY_SETTLE_RANGE)) || ((StuckCountOf(actor) - m_iSentryStuckMark[actor]) >= SENTRY_STUCK_GIVEUP))
	{
		m_iSentryStuckMark[actor] = StuckCountOf(actor);
		m_aSentryStuckArea[actor] = m_aNestArea[actor];
		m_aNestArea[actor] = PickBuildArea(actor);
		m_iSentryTry[actor] = 0;
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
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	ILocomotion myLoco = myNextbot.GetLocomotionInterface();
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
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing sentry");
	}
	if (rangeToStand > 70.0)
	{
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
		if (GetGameTime() >= m_ctSentryPressed[actor])
		{
			m_ctSentryPressed[actor] = GetGameTime() + SENTRY_PRESS_SETTLE;
			VS_PressFireButton(actor);
		}
		if (!IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctSentryTryDeadline[actor]))
		{
			m_iSentryTry[actor]++;
			if (m_iSentryTry[actor] >= SENTRY_TRY_POINTS)
			{
				m_aNestArea[actor] = PickBuildArea(actor);
				m_iSentryTry[actor] = 0;
			}
			SentryStandPoint(actor);
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
	SetPlayerReady(actor, true);
	LogBuildFailure(actor, "sentry", "built one");
	return action.Done("Built a sentry");
}

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

public void CTFBotMvMEngineerBuildSentrygun_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
	LogBuildFailure(actor, "sentry", (GetObjectOfType(actor, TFObject_Sentry) != INVALID_ENT_REFERENCE ? "ended with a sentry" : "ended with nothing"));
}

