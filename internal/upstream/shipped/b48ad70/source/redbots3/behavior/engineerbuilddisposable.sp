/* The disposable sentry, put somewhere on purpose

Nothing placed one. The upgrade was bought, the game handed the engineer a second sentry the next
time he pressed build, and it went down wherever he happened to be facing: reported from play as
minis pointing at walls and wedged into corners. GetObjectOfType skips disposable buildings
entirely, so the rest of the mod could not even see that it had happened.

It goes beside the real one now, because that is what it is for: the nest spot was chosen for what
it can see, and a second gun three metres from it sees the same ground. The ring is walked the same
way the teleporter exit walks the nest, and a point with no line to the bomb is skipped rather than
built on, which is the difference between a second gun and a second thing for a giant to break.

Between rounds only. A disposable sentry is a hundred metal and a walk, and doing either in the
middle of a wave is the engineer not repairing the sentry that matters. */

//Far enough not to be inside the real one, near enough to hold the same ground
#define DISPOSABLE_RING_RADIUS	170.0
#define DISPOSABLE_BUILD_REACH	90.0
#define DISPOSABLE_TRY_POINTS	8
#define DISPOSABLE_TRY_TIME		1.5

//Long enough to walk round the nest and try every side of it, and no longer
#define DISPOSABLE_BUILD_TIME	20.0

static float m_ctDisposableGiveUp[MAXPLAYERS + 1];
static float m_ctDisposableTryDeadline[MAXPLAYERS + 1];
static int m_iDisposableTry[MAXPLAYERS + 1];
static float m_vDisposableSpot[MAXPLAYERS + 1][3];
static float m_vDisposableStand[MAXPLAYERS + 1][3];
static bool m_bDisposableGaveUp[MAXPLAYERS + 1];

BehaviorAction CTFBotMvMEngineerBuildDisposable()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildDisposable");

	action.OnStart = CTFBotMvMEngineerBuildDisposable_OnStart;
	action.Update = CTFBotMvMEngineerBuildDisposable_Update;
	action.OnEnd = CTFBotMvMEngineerBuildDisposable_OnEnd;

	return action;
}

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
	//The wave is what the real sentry is for, and this is not worth a second of it
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		return action.Done("Wave started");

	int sentry = GetObjectOfType(actor, TFObject_Sentry);

	if (sentry == INVALID_ENT_REFERENCE)
		return action.Done("No sentry to stand one beside");

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

	float spot[3]; spot = m_vDisposableSpot[actor];
	float stand[3]; stand = m_vDisposableStand[actor];

	float range = GetVectorDistance(GetAbsOrigin(actor), stand);

	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();

	//The toolbox comes out on the way in, so arriving is not another wait
	if (range < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Sentry))
			FakeClientCommandThrottled(actor, "build 2");

		//It goes where he looks, so he looks at the spot rather than at his own feet
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, _, "Placing disposable sentry");
	}

	if (range > 70.0)
	{
		//The clock on this attempt starts when he arrives: the walk to it is not a look at it
		m_ctDisposableTryDeadline[actor] = GetGameTime() + DISPOSABLE_TRY_TIME;

		g_arrPluginBot[actor].SetPathGoalVector(stand);
		g_arrPluginBot[actor].bPathing = true;

		return action.Continue();
	}

	g_arrPluginBot[actor].bPathing = false;

	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);

	if (myWeapon != -1 && TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER)
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");

		//The toolbox is out but the game has not decided yet
		if (objBeingBuilt == -1)
			return action.Continue();

		if (!IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget()
			&& GetGameTime() > m_ctDisposableTryDeadline[actor])
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

/* Where this attempt puts it, and where he stands to put it there

Round the sentry rather than round the nest centre, because the sentry is the thing it is meant to
stand beside, and he stands between the two so the gun goes down in front of him. */
static void DisposableStandPoint(int actor)
{
	int sentry = GetObjectOfType(actor, TFObject_Sentry);

	if (sentry == INVALID_ENT_REFERENCE)
		return;

	float at[3]; at = GetAbsOrigin(sentry);

	BuildStandPoint(at, GetAbsOrigin(actor), m_iDisposableTry[actor],
		DISPOSABLE_TRY_POINTS, DISPOSABLE_RING_RADIUS, m_vDisposableSpot[actor]);

	BuildStandPoint(at, GetAbsOrigin(actor), m_iDisposableTry[actor],
		DISPOSABLE_TRY_POINTS, DISPOSABLE_RING_RADIUS - DISPOSABLE_BUILD_REACH, m_vDisposableStand[actor]);
}

public void CTFBotMvMEngineerBuildDisposable_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;

	UpdateLookAroundForEnemies(actor, true);
}

//A new wave is a new chance at ground that refused him last time
void EngineerDisposable_ForgetGivingUp()
{
	for (int i = 1; i <= MaxClients; i++)
		m_bDisposableGaveUp[i] = false;
}

//How many the upgrade he bought entitles him to, and none at all when he has not bought it
int DisposableSentriesAllowed(int client)
{
	return TF2Attrib_HookValueInt(0, "engy_disposable_sentries", client);
}

/* How many he has standing, which nothing else in the mod counts

GetObjectOfType walks past disposable buildings on purpose: everywhere else in this mod "the
sentry" means the real one, and a mini answering that question would have the engineer defending a
nest he has not built. This is the one place that wants the other answer. */
int CountDisposableSentries(int client)
{
	int count = 0;
	int objects = PlayerObjectCount(client);

	for (int i = 0; i < objects; i++)
	{
		int owned = TF2Util_GetPlayerObject(client, i);

		if (TF2_GetObjectType(owned) == TFObject_Sentry && TF2_IsDisposableBuilding(owned))
			count++;
	}

	return count;
}

/* Whether he should go and stand one beside the nest

After the nest is finished and before the wave, and only where the gun would see the ground the
real one sees: a mini behind a wall is a hundred metal and a thing for a giant to break on its way
past. */
bool ShouldBuildDisposable(int actor)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		return false;

	if (m_bDisposableGaveUp[actor])
		return false;

	if (DisposableSentriesAllowed(actor) < 1)
		return false;

	if (CountDisposableSentries(actor) >= DisposableSentriesAllowed(actor))
		return false;

	//The nest first, always: a mini is what he does with what is left over
	if (GetObjectOfType(actor, TFObject_Sentry) == INVALID_ENT_REFERENCE)
		return false;

	return GetObjectOfType(actor, TFObject_Dispenser) != INVALID_ENT_REFERENCE;
}
