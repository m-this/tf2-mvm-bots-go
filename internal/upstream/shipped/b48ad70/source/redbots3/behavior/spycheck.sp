/* Spy checking, which is a thing a bot can do without being told where the Spy is

The idea and the shape of it are Cheeseh's, from RCBot2's CSpyCheckAir and the paranoia that
starts it. It is worth saying what makes it good, because a first attempt at this problem is
always to look up the Spy's position and shoot at it, and that is not a bot playing, it is a bot
cheating quietly.

Two pieces, and neither of them reads anything a player could not see.

The first is paranoia. A team that has seen a Spy knows there is a Spy, and it does not know
where he went. So the moment one is seen, the position and the time are remembered, and a circle
around that spot grows at a walking pace. A bot inside the circle is a bot the Spy could have
reached by now, and it starts checking. A bot outside it gets on with the wave. That is why the
bots stop being paranoid on their own, and why a Spy who works one end of the map does not put
the other end on alert.

The second is the tell, and it is the good part. A Spy is wearing a face, so looking at the face
is worthless. What gives him away is that he was not there a moment ago. The bot takes a list of
the teammates it can see, and then watches for one that appears in view who was not in the list.
A real teammate walks into view from somewhere; a Spy tends to already be next to you.

Then it hits him. That costs nothing to be wrong about: friendly fire is off, so a swing at a
real teammate does nothing at all, and a swing at a disguised Spy hurts him and breaks the
disguise. And it stops early if the suspect fires his own weapon, because a robot being shot at
by the suspect is the alibi a Spy cannot produce.

None of this makes a bot unstabbable, which is the point. A Spy who waits for the check to end
still gets his stab. */

//How fast the circle of ground a Spy could have reached grows, in units a second
#define SPY_PARANOIA_SPEED	320.0

//How far the paranoia reaches at most, however long ago the Spy was seen
#define SPY_PARANOIA_RANGE_MAX	2000.0

//How long a sighting is worth anything at all
#define SPY_PARANOIA_MEMORY	20.0

//A check is seconds out of a wave, so it is bounded at both ends
#define SPY_CHECK_MIN_TIME	2.0
#define SPY_CHECK_MAX_TIME	5.0

//Melee range, near enough. Past this the bot walks in rather than swinging at nothing
#define SPY_CHECK_REACH		80.0

//How often the bot looks for somebody who was not there before
#define SPY_CHECK_LOOK_INTERVAL	0.1

//Knife range, and a little more, for the Spy who is simply standing behind the bot
#define SPY_BEHIND_RANGE	400.0

//How long he has to be back there. Instant would be a bot that cannot be flanked at all
#define SPY_BEHIND_TIME		0.2

/* Where and when this team last saw a Spy

One position for the whole team, not one per bot. A Spy is seen by somebody, and a team that has
seen one talks about it. Splitting it per bot would mean the Spy has to be seen six times before
the team reacts, which is a worse model of a team than a shared one */
float g_flLastSpySeenTime;
float g_vLastSpySeen[3];

static float m_ctSpyCheckEnd[MAXPLAYERS + 1];
static float m_ctSpyCheckNextLook[MAXPLAYERS + 1];
static int m_iSpyCheckSuspect[MAXPLAYERS + 1];
static bool m_bSpyCheckHit[MAXPLAYERS + 1];
static bool m_bSpyCheckSeen[MAXPLAYERS + 1][MAXPLAYERS + 1];
static float m_ctSpyBehindSince[MAXPLAYERS + 1];

//A Spy was seen doing something a Spy does. Everything else here follows from this being called
void NoteSpySighting(const float origin[3])
{
	g_flLastSpySeenTime = GetGameTime();
	g_vLastSpySeen = origin;
}

void ResetSpyIntel()
{
	g_flLastSpySeenTime = 0.0;
	g_vLastSpySeen = NULL_VECTOR;
}

/* The ground a Spy could have covered since he was last seen

Grows with time and stops growing at the maximum, so an old sighting eventually means the whole
area is suspect, and then the memory runs out and none of it is */
static bool IsInSpyParanoiaRange(int client)
{
	if (g_flLastSpySeenTime <= 0.0)
		return false;

	float elapsed = GetGameTime() - g_flLastSpySeenTime;

	if (elapsed > SPY_PARANOIA_MEMORY)
		return false;

	float reach = MinFloat(elapsed * SPY_PARANOIA_SPEED, SPY_PARANOIA_RANGE_MAX);

	float myOrigin[3]; GetClientAbsOrigin(client, myOrigin);

	return GetVectorDistance(myOrigin, g_vLastSpySeen) <= reach;
}

BehaviorAction CTFBotSpyCheck()
{
	BehaviorAction action = ActionsManager.Create("DefenderSpyCheck");

	action.OnStart = CTFBotSpyCheck_OnStart;
	action.Update = CTFBotSpyCheck_Update;

	return action;
}

public Action CTFBotSpyCheck_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));

	m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(SPY_CHECK_MIN_TIME, SPY_CHECK_MAX_TIME);
	m_ctSpyCheckNextLook[actor] = 0.0;
	m_iSpyCheckSuspect[actor] = -1;
	m_bSpyCheckHit[actor] = false;

	//The teammates that are already there. Anybody who turns up after this is the one worth hitting
	SnapshotVisibleTeammates(actor);

	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_CLOAKEDSPY);

	return action.Continue();
}

public Action CTFBotSpyCheck_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_ctSpyCheckEnd[actor] < GetGameTime())
		return action.Done("Checked for long enough");

	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);

	//A robot in front of the bot outranks any suspicion about a teammate behind it
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(true);

	if (threat != NULL_KNOWN_ENTITY)
		return action.Done("Something real to shoot at");

	int suspect = m_iSpyCheckSuspect[actor];

	if (IsValidClientIndex(suspect) && !IsPlayerAlive(suspect))
		suspect = -1;

	if (suspect == -1)
	{
		if (m_ctSpyCheckNextLook[actor] < GetGameTime())
		{
			m_ctSpyCheckNextLook[actor] = GetGameTime() + SPY_CHECK_LOOK_INTERVAL;

			suspect = FindTeammateWhoWasNotThere(actor);

			//Somebody turned up, so the check is worth a little longer than it had left
			if (suspect != -1)
				m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(SPY_CHECK_MIN_TIME, SPY_CHECK_MAX_TIME);
		}

		m_iSpyCheckSuspect[actor] = suspect;
	}

	if (suspect == -1)
		return action.Continue();

	/* He is shooting at something, so he is not a Spy
	The one alibi a disguised Spy cannot produce: his weapon is a knife wearing somebody else's
	model, and firing it drops the disguise */
	if (GetTimeSinceWeaponFired(suspect) < 1.0)
	{
		m_iSpyCheckSuspect[actor] = -1;

		return action.Done("The suspect is fighting");
	}

	IBody myBody = myBot.GetBodyInterface();

	AimHeadTowards(myBody, WorldSpaceCenter(suspect), CRITICAL, 0.5, _, "Spy check");

	float range = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(suspect));

	if (range > SPY_CHECK_REACH)
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToTarget(actor, myBot, suspect);
		}

		m_pPath[actor].Update(myBot);
	}

	/* Swing. Friendly fire is off, so being wrong about this costs nothing at all, and being
	right takes the disguise off him */
	if (myBody.IsHeadAimingOnTarget())
		VS_PressFireButton(actor);

	if (range < SPY_CHECK_REACH)
	{
		//Hit him once and move on. A bot that stands there hitting a teammate is a bot not playing
		if (!m_bSpyCheckHit[actor])
		{
			m_bSpyCheckHit[actor] = true;
			m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(0.5, 1.5);
		}
	}

	return action.Continue();
}

//Everybody on the bot's own team it can see right now
static void SnapshotVisibleTeammates(int actor)
{
	IVision myVision = CBaseNPC_GetNextBotOfEntity(actor).GetVisionInterface();
	TFTeam myTeam = TF2_GetClientTeam(actor);

	for (int i = 1; i <= MaxClients; i++)
	{
		m_bSpyCheckSeen[actor][i] = IsClientInGame(i) && i != actor && IsPlayerAlive(i)
			&& TF2_GetClientTeam(i) == myTeam && myVision.IsAbleToSeeTarget(i, USE_FOV);
	}
}

/* A teammate in view who was not in view when the check started, or -1

The whole tell. It is recorded as seen the moment it is returned, so the same teammate is not
suspected twice in one check and the bot moves on to whoever else turns up */
static int FindTeammateWhoWasNotThere(int actor)
{
	IVision myVision = CBaseNPC_GetNextBotOfEntity(actor).GetVisionInterface();
	TFTeam myTeam = TF2_GetClientTeam(actor);

	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == actor || !IsClientInGame(i) || !IsPlayerAlive(i))
			continue;

		if (TF2_GetClientTeam(i) != myTeam)
			continue;

		/* A human teammate is never the disguised one
		
		Every robot in this mode is a fake client, so a real player on RED cannot be an enemy Spy,
		and frisking him for it is noise he can see: reported from play as the team calling a player
		out as a Spy and shooting at him while he was trying to play one.
		
		It costs the one case where a human is on BLU through the mod's own join-blue command and
		has disguised as a defender. That is a curiosity, and being unstabbable in it is a smaller
		price than a Spy player being shot by his own team every wave. */
		if (!IsFakeClient(i))
		{
			m_bSpyCheckSeen[actor][i] = true;
			continue;
		}

		if (m_bSpyCheckSeen[actor][i])
			continue;

		if (!myVision.IsAbleToSeeTarget(i, USE_FOV))
			continue;

		//Whoever is carrying the bomb is not a disguise, because a Spy carrying it is not disguised
		if (TF2_HasTheFlag(i))
		{
			m_bSpyCheckSeen[actor][i] = true;
			continue;
		}

		m_bSpyCheckSeen[actor][i] = true;

		return i;
	}

	return -1;
}

/* What this bot can honestly claim to have seen of a Spy, fed to the team's memory of one

A disguised Spy is deliberately not a sighting. He is wearing a face and the bot believes the
face, which is the whole contract of the disguise and the reason the tell above is worth having.
What counts is a Spy with no disguise on, a Spy whose cloak has been broken, and, separately, a
Spy standing at the bot's back.

The last one is not RCBot2's and it is not paranoia. A player who has somebody at knife distance
behind him for half a second turns around, whatever he believes about who it is */
/* Looking behind you, which is the whole of Spy defence a bot never did

A player on a Spy wave turns round. The bot did not: it noticed a Spy only once one had stood
within SPY_BEHIND_RANGE of its back for SPY_BEHIND_TIME, which on a wave of a hundred Spies is
a description of being stabbed rather than a defence against it.

So while the team is worried about Spies, a bot turns and looks. The glance is short and on a
loose interval: a bot that spins constantly never shoots anything, and one that spins on a fixed
beat is a bot a Spy walks in behind between beats. */
#define SPY_GLANCE_INTERVAL_MIN	1.6
#define SPY_GLANCE_INTERVAL_MAX	3.2
#define SPY_GLANCE_TIME			0.35
#define SPY_GLANCE_RANGE		220.0

static float m_ctNextSpyGlance[MAXPLAYERS + 1];

static void UpdateSpyGlance(int client)
{
	if (!Feature(FEATURE_SPY_GLANCE) || !IsInSpyParanoiaRange(client))
	{
		m_ctNextSpyGlance[client] = 0.0;
		return;
	}

	INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);

	//Something in front is a better use of the eyes than something that might be behind
	if (myBot.GetVisionInterface().GetPrimaryKnownThreat(true) != NULL_KNOWN_ENTITY)
		return;

	if (m_ctNextSpyGlance[client] > GetGameTime())
		return;

	m_ctNextSpyGlance[client] = GetGameTime() + GetRandomFloat(SPY_GLANCE_INTERVAL_MIN, SPY_GLANCE_INTERVAL_MAX);

	float myAngles[3]; GetClientEyeAngles(client, myAngles);
	float myForward[3]; GetAngleVectors(myAngles, myForward, NULL_VECTOR, NULL_VECTOR);

	float behind[3]; behind = GetEyePosition(client);
	behind[0] -= myForward[0] * SPY_GLANCE_RANGE;
	behind[1] -= myForward[1] * SPY_GLANCE_RANGE;
	behind[2] -= myForward[2] * SPY_GLANCE_RANGE;

	AimHeadTowards(myBot.GetBodyInterface(), behind, IMPORTANT, SPY_GLANCE_TIME, _, "Checking behind me");
}

void UpdateSpyIntel(int client)
{
	UpdateSpyGlance(client);

	IVision myVision = CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface();
	TFTeam enemyTeam = GetPlayerEnemyTeam(client);

	float myOrigin[3]; GetClientAbsOrigin(client, myOrigin);
	float myAngles[3]; GetClientEyeAngles(client, myAngles);
	float myForward[3]; GetAngleVectors(myAngles, myForward, NULL_VECTOR, NULL_VECTOR);

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i))
			continue;

		if (TF2_GetClientTeam(i) != enemyTeam || TF2_GetPlayerClass(i) != TFClass_Spy)
			continue;

		//Cloak is cloak. A bot that sees through it is a bot no Spy can play against
		if (TF2_IsStealthed(i) && !IsCloakedPlayerExposed(i))
			continue;

		float theirOrigin[3]; GetClientAbsOrigin(i, theirOrigin);

		if (GetVectorDistance(myOrigin, theirOrigin) <= SPY_BEHIND_RANGE)
		{
			float toThem[3]; SubtractVectors(theirOrigin, myOrigin, toThem);
			NormalizeVector(toThem, toThem);

			//Behind, which is anything the bot is not roughly facing
			if (GetVectorDotProduct(myForward, toThem) < 0.0)
			{
				if (m_ctSpyBehindSince[client] <= 0.0)
					m_ctSpyBehindSince[client] = GetGameTime();
				else if (GetGameTime() - m_ctSpyBehindSince[client] >= SPY_BEHIND_TIME)
				{
					m_ctSpyBehindSince[client] = 0.0;

					myVision.AddKnownEntity(i);
					NoteSpySighting(theirOrigin);
				}

				continue;
			}
		}

		//Nothing pretending to be anything, in plain view. That is a sighting
		if (!TF2_IsPlayerInCondition(i, TFCond_Disguised) && myVision.IsAbleToSeeTarget(i, USE_FOV))
			NoteSpySighting(theirOrigin);
	}
}

void ResetSpyCheck(int client)
{
	m_ctSpyBehindSince[client] = 0.0;
	m_ctSpyCheckEnd[client] = 0.0;
	m_iSpyCheckSuspect[client] = -1;
}

bool CTFBotSpyCheck_IsPossible(int client)
{
	if (!IsPlayerAlive(client) || TF2_IsInUpgradeZone(client))
		return false;

	//A bot in the middle of a fight has better things to do than frisk its own team
	if (CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface().GetPrimaryKnownThreat(true) != NULL_KNOWN_ENTITY)
		return false;

	/* An engineer holding a nest is doing the one job nobody else can do, and the sentry is the
	spy check: anything that walks into it while sapping is already being shot at */
	if (TF2_GetPlayerClass(client) == TFClass_Engineer)
		return false;

	return IsInSpyParanoiaRange(client);
}
