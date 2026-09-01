BehaviorAction CTFBotMvMEngineerBuildDisposable()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildDisposable");

	action.OnStart = CTFBotMvMEngineerBuildDisposable_OnStart;
	action.Update = CTFBotMvMEngineerBuildDisposable_Update;
	action.OnEnd = CTFBotMvMEngineerBuildDisposable_OnEnd;

	return action;
}

#define Go_Slots (65)

#define DISPOSABLE_RING_RADIUS (170.0)
#define DISPOSABLE_BUILD_REACH (90.0)
#define DISPOSABLE_TRY_POINTS (8)
#define DISPOSABLE_TRY_TIME (1.5)

#define DISPOSABLE_BUILD_TIME (20.0)

float m_ctDisposableGiveUp[65];
float m_ctDisposableTryDeadline[65];
int m_iDisposableTry[65];
float m_vDisposableSpot[65][3];
float m_vDisposableStand[65][3];
bool m_bDisposableGaveUp[65];

public Action CTFBotMvMEngineerBuildDisposable_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_ctDisposableGiveUp[actor] = GetGameTime() + DISPOSABLE_BUILD_TIME;
	m_ctDisposableTryDeadline[actor] = GetGameTime() + DISPOSABLE_TRY_TIME;
	m_iDisposableTry[actor] = 0;
	DisposableStandPoint(actor);
	UpdateLookAroundForEnemies(actor, true);
	return action.Continue();
}

public Action CTFBotMvMEngineerBuildDisposable_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return action.Done("Wave started");
	}
	int sentry = GetObjectOfType(actor, TFObject_Sentry);
	if (sentry == INVALID_ENT_REFERENCE)
	{
		return action.Done("No sentry to stand one beside");
	}
	if (CountDisposableSentries(actor) >= DisposableSentriesAllowed(actor))
	{
		g_arrPluginBot[actor].bPathing = false;
		return action.Done("Built one");
	}
	if (m_ctDisposableGiveUp[actor] < GetGameTime())
	{
		m_bDisposableGaveUp[actor] = true;
		return action.Done("Nowhere beside the nest will take one");
	}
	float mySpot[3];
	mySpot = m_vDisposableSpot[actor];
	float myStand[3];
	myStand = m_vDisposableStand[actor];
	float buildRange = GetVectorDistance(GetAbsOrigin(actor), myStand);
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	if (buildRange < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Sentry))
		{
			FakeClientCommandThrottled(actor, "build 2");
		}
		AimHeadTowards(myBody, mySpot, MANDATORY, 0.1, Address_Null, "Placing disposable sentry");
	}
	if (buildRange > 70.0)
	{
		m_ctDisposableTryDeadline[actor] = GetGameTime() + DISPOSABLE_TRY_TIME;
		g_arrPluginBot[actor].SetPathGoalVector(myStand);
		g_arrPluginBot[actor].bPathing = true;
		return action.Continue();
	}
	g_arrPluginBot[actor].bPathing = false;
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	if ((myWeapon != -1) && (TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER))
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");
		if (objBeingBuilt == -1)
		{
			return action.Continue();
		}
		if (!IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctDisposableTryDeadline[actor]))
		{
			m_iDisposableTry[actor]++;
			if (m_iDisposableTry[actor] >= DISPOSABLE_TRY_POINTS)
			{
				m_bDisposableGaveUp[actor] = true;
				return action.Done("Every side of the nest refused one");
			}
			DisposableStandPoint(actor);
			return action.Continue();
		}
	}
	VS_PressFireButton(actor);
	return action.Continue();
}

stock void DisposableStandPoint(int actor)
{
	int sentry = GetObjectOfType(actor, TFObject_Sentry);
	if (sentry == INVALID_ENT_REFERENCE)
	{
		return;
	}
	float at[3];
	at = GetAbsOrigin(sentry);
	BuildStandPoint(at, GetAbsOrigin(actor), m_iDisposableTry[actor], DISPOSABLE_TRY_POINTS, DISPOSABLE_RING_RADIUS, m_vDisposableSpot[actor]);
	BuildStandPoint(at, GetAbsOrigin(actor), m_iDisposableTry[actor], DISPOSABLE_TRY_POINTS, 80.0, m_vDisposableStand[actor]);
}

public void CTFBotMvMEngineerBuildDisposable_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
}

stock void EngineerDisposable_ForgetGivingUp()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_bDisposableGaveUp[i] = false;
	}
}

stock int DisposableSentriesAllowed(int client)
{
	return TF2Attrib_HookValueInt(0, "engy_disposable_sentries", client);
}

stock int CountDisposableSentries(int client)
{
	int count = 0;
	int objects = PlayerObjectCount(client);
	for (int i = 0; i < objects; i++)
	{
		int owned = TF2Util_GetPlayerObject(client, i);
		if ((TF2_GetObjectType(owned) == TFObject_Sentry) && TF2_IsDisposableBuilding(owned))
		{
			count++;
		}
	}
	return count;
}

stock bool ShouldBuildDisposable(int actor)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return false;
	}
	if (m_bDisposableGaveUp[actor])
	{
		return false;
	}
	if (DisposableSentriesAllowed(actor) < 1)
	{
		return false;
	}
	if (CountDisposableSentries(actor) >= DisposableSentriesAllowed(actor))
	{
		return false;
	}
	if (GetObjectOfType(actor, TFObject_Sentry) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	return GetObjectOfType(actor, TFObject_Dispenser) != INVALID_ENT_REFERENCE;
}

