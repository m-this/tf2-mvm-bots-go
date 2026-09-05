BehaviorAction CTFBotMvMEngineerBuildDisposable()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildDisposable");

	action.OnStart = CTFBotMvMEngineerBuildDisposable_OnStart;
	action.Update = CTFBotMvMEngineerBuildDisposable_Update;
	action.OnEnd = CTFBotMvMEngineerBuildDisposable_OnEnd;

	return action;
}

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

// OnStart starts the clock and picks the first side of the nest to try.
public Action CTFBotMvMEngineerBuildDisposable_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_ctDisposableGiveUp[actor] = GetGameTime() + DISPOSABLE_BUILD_TIME;
	m_ctDisposableTryDeadline[actor] = GetGameTime() + DISPOSABLE_TRY_TIME;
	m_iDisposableTry[actor] = 0;
	DisposableStandPoint(actor);
	UpdateLookAroundForEnemies(actor, true);
	return action.Continue();
}

// Update walks to the spot, gets the toolbox out on the way, and tries the next
// side of the nest when the game refuses this one.
public Action CTFBotMvMEngineerBuildDisposable_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	// The wave is what the real sentry is for, and this is not worth a second of it
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
	// The toolbox comes out on the way in, so arriving is not another wait
	if (buildRange < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Sentry))
		{
			FakeClientCommandThrottled(actor, "build 2");
		}
		// It goes where he looks, so he looks at the spot rather than at his own feet
		AimHeadTowards(myBody, mySpot, MANDATORY, 0.1, Address_Null, "Placing disposable sentry");
	}
	if (buildRange > 70.0)
	{
		// The clock on this attempt starts when he arrives: the walk to it is not a look at it
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
		// The toolbox is out but the game has not decided yet
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

// StandPoint is where this attempt puts it, and where he stands to put it there.
//
// Round the sentry rather than round the nest centre, because the sentry is the
// thing it is meant to stand beside, and he stands between the two so the gun goes
// down in front of him.
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

// OnEnd stops the walking and gives the looking back.
public void CTFBotMvMEngineerBuildDisposable_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
}

// ForgetGivingUp is a new wave being a new chance at ground that refused him
// last time.
stock void EngineerDisposable_ForgetGivingUp()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_bDisposableGaveUp[i] = false;
	}
}

// DisposableSentriesAllowed is how many the upgrade he bought entitles him to,
// and none at all when he has not bought it.
stock int DisposableSentriesAllowed(int client)
{
	return TF2Attrib_HookValueInt(0, "engy_disposable_sentries", client);
}

// CountDisposableSentries is how many he has standing, which nothing else in the
// mod counts.
//
// GetObjectOfType walks past disposable buildings on purpose: everywhere else in
// this mod "the sentry" means the real one, and a mini answering that question
// would have the engineer defending a nest he has not built. This is the one place
// that wants the other answer.
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

// ShouldBuildDisposable is whether he should go and stand one beside the nest.
//
// After the nest is finished and before the wave, and only where the gun would see
// the ground the real one sees: a mini behind a wall is a hundred metal and a thing
// for a giant to break on its way past.
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
	// The nest first, always: a mini is what he does with what is left over
	if (GetObjectOfType(actor, TFObject_Sentry) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	return GetObjectOfType(actor, TFObject_Dispenser) != INVALID_ENT_REFERENCE;
}

