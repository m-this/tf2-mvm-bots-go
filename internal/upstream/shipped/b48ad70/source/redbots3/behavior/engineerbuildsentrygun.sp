/* The sentry, which is the whole of the engineer's job and was the last building still guessing

The dispenser and the teleporter both learned the same lesson and this had not: a building goes
down in front of the man and never under him, so walking onto the spot and pressing fire aims the
sentry at whatever is a build's reach beyond it. The old code walked to the nest point, stood on
it, and aimed at its own feet. Between rounds that mostly worked, because the engineer is
teleported onto the point and the ground under a nest hint is usually clear; in the middle of a
wave, having walked there, it did not.

There was also no clock on any of it. No reach deadline, no give-up: an engineer who could not
place a sentry stayed in this action for the rest of the wave, which is what a test-bed run of
Bigrock's first wave looked like from outside. Eight minutes, no sentry, and nothing in the logs
saying why. Everything here has a limit now, and running out of one hands the engineer back to the
idle action, which tries again three seconds later with a freshly scored nest. */

//A build's reach short of the spot, with the spot in front of him, same as the other two
#define SENTRY_BUILD_REACH	90.0

/* Eight looks at the spot, one from each side, before the spot itself is the thing in question

A sentry refused from one side is usually a sentry with a wall behind it rather than a sentry on
bad ground, and the answer to that is to stand somewhere else. Re-scoring the nest on the first
refusal, which is what this did, threw away a good spot for a bad reason and cost a full pass over
the nav mesh every time it happened. */
#define SENTRY_TRY_POINTS	8
//Long enough for the game to act on a press, short enough to retry one it refused
#define SENTRY_PRESS_SETTLE		0.3

#define SENTRY_TRY_TIME		1.5

/* How long the walk and the whole business may take

The walk is priced by its length, because "the walk is inside the nest" stopped being true the
moment he started every one of them at the upgrade station. Past the build time he goes back to
the idle action rather than settling for where he stands: a sentry is not a dispenser, and one
pointed at a wall is worse than three more seconds spent finding somewhere it can see from.

The settle range is the important one. Running out of clock used to mean building beside himself
wherever he had got to, with no distance test of any kind: that is a sentry at a random place on
the map, reported from play on Coaltown, and this file's own comment admits to one 625 units from
its nest on Decoy. Two build reaches is close enough that what he settles for still sees what the
nest was chosen to see. Further out he keeps walking, and the give-up clock hands him back to the
idle action, which scores a nest again and tries afresh. */
#define SENTRY_REACH_TIME	12.0
#define SENTRY_SETTLE_RANGE	200.0
#define SENTRY_BUILD_TIME	45.0

static float m_ctSentryReachDeadline[MAXPLAYERS + 1];

/* What the stuck watchdog had counted when this build attempt started

The watchdog resets the whole behaviour every STUCK_TIME, which restarts this action, which re-arms
the reach deadline. So an engineer who cannot reach his spot is rescued by nothing: the deadline
that would have made him pick another spot is pushed forward every time he is rescued.

Measured on Mannworks with Mean Machines: the same engineer stuck at 1014 885 274, nineteen times,
inside DefenderBuildSentrygun, and never a sentry. STUCK_TIME and SENTRY_REACH_TIME are both twelve
seconds, so the two timers chase each other.

Counting stucks across the restarts is what survives them. */
static int m_iSentryStuckMark[MAXPLAYERS + 1];

//The nest the mark above belongs to, so a restart does not look like a new attempt
static CNavArea m_aSentryStuckArea[MAXPLAYERS + 1] = {NULL_AREA, ...};

//Stucks inside one attempt before the spot is the suspect rather than the walk
#define SENTRY_STUCK_GIVEUP	2
static float m_ctSentryGiveUpTime[MAXPLAYERS + 1];
static float m_ctSentryTryDeadline[MAXPLAYERS + 1];
//When the last build press is allowed to have landed, so the next frame is not another press
static float m_ctSentryPressed[MAXPLAYERS + 1];
static int m_iSentryTry[MAXPLAYERS + 1];
static float m_vSentrySpot[MAXPLAYERS + 1][3];
static float m_vSentryStand[MAXPLAYERS + 1][3];

BehaviorAction CTFBotMvMEngineerBuildSentrygun()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildSentrygun");
	
	action.OnStart = CTFBotMvMEngineerBuildSentrygun_OnStart;
	action.Update = CTFBotMvMEngineerBuildSentrygun_Update;
	action.OnEnd = CTFBotMvMEngineerBuildSentrygun_OnEnd;
	
	return action;
}

public Action CTFBotMvMEngineerBuildSentrygun_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
	
	m_ctSentryGiveUpTime[actor] = GetGameTime() + SENTRY_BUILD_TIME;
	m_ctSentryTryDeadline[actor] = GetGameTime() + SENTRY_TRY_TIME;
	m_ctSentryPressed[actor] = 0.0;
	m_iSentryTry[actor] = 0;
	
	if (GameRules_GetRoundState() == RoundState_BetweenRounds)
	{
		if (m_aNestArea[actor])
		{
			//Teleport ourselves to the nest area for a faster setup
			float vNestPosition[3]; NestBuildPosition(m_aNestArea[actor], vNestPosition);
			vNestPosition[2] += TFBOT_STEP_HEIGHT;
			CBaseEntity(actor).SetAbsOrigin(vNestPosition);
		}
	}
	
	SentryStandPoint(actor);
	
	//After the teleport above, so a between-rounds walk is priced from where he actually starts it
	m_ctSentryReachDeadline[actor] = GetGameTime() + BuildReachTime(GetAbsOrigin(actor), m_vSentryStand[actor]);
	
	LogBuildFailure(actor, "sentry", "started");

	/* The mark has to survive the watchdog's reset, which is the whole point of it

	ResetIntentionInterface restarts this action, so anything armed in OnStart is armed again every
	twelve seconds and can never expire. That is the fault in the reach deadline, and re-marking
	here would inherit it: the count would restart alongside the thing it counts.

	So the mark belongs to the nest rather than to the attempt. It resets when he is sent somewhere
	new, and not when he is merely restarted at the same place. */
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
		//And you.
		
		LogBuildFailure(actor, "sentry", "told to advance the nest");
		return action.Done("No sentry");
	}
	
	//Every side of this spot refused him and the walk is not getting shorter. The idle action retries
	if (GetGameTime() > m_ctSentryGiveUpTime[actor])
	{
		LogBuildFailure(actor, "sentry", "every side of the spot refused him");
		
		return action.Done("Nowhere here will take a sentry");
	}
	
	float spot[3]; spot = m_vSentrySpot[actor];
	float stand[3]; stand = m_vSentryStand[actor];
	
	/* The walk ran out, so he builds from where he got to rather than into whatever stopped him
	
	And he puts it beside himself rather than pointing it at the nest he could not reach. Aiming at
	the nest from three metres short of it is the same thing; aiming at it from twenty metres short
	puts the sentry twenty metres from where anybody wanted it, facing a direction chosen by where
	he happened to get stuck. Decoy produced one 625 units from its own nest that way. */
	float range_to_spot = GetVectorDistance(GetAbsOrigin(actor), m_vSentrySpot[actor]);
	bool outOfTime = GetGameTime() > m_ctSentryReachDeadline[actor] && range_to_spot < SENTRY_SETTLE_RANGE;

	/* The walk ran out and he is nowhere near the spot, so the spot is what to give up on

	outOfTime above deliberately refuses to build from far away, for the reason in the comment on
	it. What that leaves is the case nothing handled: an engineer who never arrives keeps walking at
	a spot he cannot reach, for the whole mission, and builds nothing at all. Reported on Mannworks
	with Mean Machines, and Bigrock has a spot on a rock he cannot jump onto.

	The retry below re-scores the nest, and it only runs once he is close enough to try building. So
	the same thing is done here, from the other side of the range check: a new area rather than a
	sentry twenty metres from where anybody wanted one. */
	if ((GetGameTime() > m_ctSentryReachDeadline[actor] && range_to_spot >= SENTRY_SETTLE_RANGE)
		|| StuckCountOf(actor) - m_iSentryStuckMark[actor] >= SENTRY_STUCK_GIVEUP)
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
		
		BuildStandPoint(stand, m_vSentrySpot[actor], m_iSentryTry[actor],
			SENTRY_TRY_POINTS, SENTRY_BUILD_REACH, spot);
	}
	
	float range_to_stand = GetVectorDistance(GetAbsOrigin(actor), stand);
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	ILocomotion myLoco = myNextbot.GetLocomotionInterface();
	
	if (range_to_stand < 200.0) 
	{
		//Start building a sentry
		if (!IsBuilderSetTo(actor, TFObject_Sentry))
			FakeClientCommandThrottled(actor, "build 2");
		
		UpdateLookAroundForEnemies(actor, false);
		
		if (!myLoco.IsStuck())
		{
			g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
		}
		
		//It goes where he looks, so he looks at the spot rather than at the ground under himself
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, _, "Placing sentry");
	}
	
	if (range_to_stand > 70.0)
	{
		//The clock on this attempt starts when he arrives: the walk to it is not a look at it
		m_ctSentryTryDeadline[actor] = GetGameTime() + SENTRY_TRY_TIME;
		
		g_arrPluginBot[actor].SetPathGoalVector(stand);
		g_arrPluginBot[actor].bPathing = true;
		
		if (range_to_stand > 300.0)
		{
			//Fuck em up.
			EquipWeaponSlot(actor, TFWeaponSlot_Primary);
		}
		
		UpdateLookAroundForEnemies(actor, true);
		
		return action.Continue();
	}
	
	g_arrPluginBot[actor].bPathing = false;
	
	if (myWeapon != -1 && TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER)
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");
		
		if (objBeingBuilt == -1)
			return action.Continue();
		
		/* One press, then a tick for the game to act on it
		
		The check at the end of this function runs in the same frame as this press, so it asks
		whether a sentry exists before the game has put one down. It answered no, the action
		carried on, and the toolbox re-armed: another press, another building. Measured on the
		dispenser, which has the same shape and which the test-bed caught standing twice under one
		engineer. */
		if (GetGameTime() >= m_ctSentryPressed[actor])
		{
			m_ctSentryPressed[actor] = GetGameTime() + SENTRY_PRESS_SETTLE;
			
			VS_PressFireButton(actor);
		}
		
		/* The game says no from here, so try looking at it from the next side round
		
		Only once he is actually looking at it: the answer while his head is still coming round is
		the answer for wherever it was pointing, which is not this spot. */
		if (!IsPlacementOK(objBeingBuilt) && myBody.IsHeadAimingOnTarget()
			&& GetGameTime() > m_ctSentryTryDeadline[actor])
		{
			m_iSentryTry[actor]++;
			
			/* Every side refused him, so now the spot itself is the thing in question
			
			This is where the nest gets re-scored, and not before: a pass over the nav mesh is the
			expensive answer and it was being given to a wall behind the man. */
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
		return action.Continue();
	
	SetPlayerReady(actor, true);
	
	LogBuildFailure(actor, "sentry", "built one");
	
	return action.Done("Built a sentry");
}

/* Where the sentry goes and where he stands to put it there, on a side he can stand on

Sides with nothing walkable under them are skipped rather than walked at: a nest on raised ground
has thin air around it, and pathing at a coordinate in mid-air puts the engineer on the floor below
holding the toolbox until a clock saves him. Bounded by the number of sides there are. */
static void SentryStandPoint(int actor)
{
	NestBuildPosition(m_aNestArea[actor], m_vSentrySpot[actor]);
	
	for (int skipped = 0; skipped < SENTRY_TRY_POINTS; skipped++)
	{
		if (BuildStandPoint(m_vSentrySpot[actor], GetAbsOrigin(actor), m_iSentryTry[actor],
			SENTRY_TRY_POINTS, SENTRY_BUILD_REACH, m_vSentryStand[actor]))
			return;
		
		m_iSentryTry[actor] = (m_iSentryTry[actor] + 1) % SENTRY_TRY_POINTS;
	}
}

public void CTFBotMvMEngineerBuildSentrygun_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	
	UpdateLookAroundForEnemies(actor, true);
	
	/* Every way out of this action, including the ones nobody wrote a branch for
	
	The Done branches above name why they gave up, and a session produced far more starts than
	endings that said anything. Asking the result for its reason here printed nothing at all, which
	is what a thrown native looks like from the outside: it takes the callback with it. So this says
	only what is certainly true, which is that the attempt is over and whether it left a sentry. */
	LogBuildFailure(actor, "sentry",
		GetObjectOfType(actor, TFObject_Sentry) != INVALID_ENT_REFERENCE ? "ended with a sentry" : "ended with nothing");
}
