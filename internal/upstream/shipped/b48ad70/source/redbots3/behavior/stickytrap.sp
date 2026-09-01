/* Laying a stickybomb trap, which is the half of the Demoman that firing the launcher is not

The state machine is Cheeseh's, from RCBot2's CBotTF2::deployStickies, and it is worth copying
because it is the small honest version of a thing that invites a large dishonest one. A bot does
not need to know where the robots will walk. It needs a piece of ground, a handful of bombs
stacked on it, a gap between shots so the launcher keeps up, and somebody to decide when the
ground is worth it.

  the ground     for a defender, where the bomb is. Robots escort it, so it is the one place on
                 the map they are all walking to, and it is where the carrier will stand
  the stack      a small scatter around one point, so a giant standing on it takes all eight
                 bombs rather than the two it walked near
  the gap        a second or so between shots, which is what the launcher wants and what stops
                 the bot emptying a clip into a wall while it turns
  the deadline   because a Demoman standing in the open aiming at the floor is a Demoman not
                 fighting, and the wave does not wait for him

Detonation is not here. ShouldDetonateStickies reads where the bombs actually landed, which is
better than trusting where they were aimed. */

//The launcher holds eight, and eight is the trap
#define STICKY_TRAP_BOMBS	8

/* How wide to scatter them, and why it is this narrow

It was 120, roughly a blast across, on the reasoning that a spread covers ground without gaps.
That is the right trap for a crowd and the wrong one for what actually walks into it here.

Every guide written about this class says the same thing: stack the bombs on one spot for a
giant, and carpet only for a group or a line of Medics. A giant is what a trap is for, because a
giant is what the team cannot kill any other way, and a giant standing on eight stacked bombs
takes all eight. The same giant walking over a carpet takes the two or three it happens to be
near, which is a giant that lives.

Forty is a stack with enough scatter that a bomb landing on a lip or a step does not take the
whole trap with it. */
#define STICKY_TRAP_SPREAD	40.0

//What it was before, and what the switch goes back to: a carpet for a crowd
#define STICKY_TRAP_CARPET	120.0

//What the launcher wants between shots, and what the bot needs to turn between them
#define STICKY_TRAP_SHOT_GAP_MIN	0.6
#define STICKY_TRAP_SHOT_GAP_MAX	0.9

//A trap is worth this many seconds of a wave and no more
#define STICKY_TRAP_MAX_TIME	12.0

//Nearer than this and the bot is standing in its own trap. Further and it is aiming at a rumour
#define STICKY_TRAP_MIN_RANGE	350.0
#define STICKY_TRAP_MAX_RANGE	1500.0

//Bombs already down that make another trap a waste of the clip
#define STICKY_TRAP_ENOUGH	4

//How long before the same bot bothers again, so a Demoman is not permanently gardening
#define STICKY_TRAP_COOLDOWN	20.0

static float m_ctStickyTrapEnd[MAXPLAYERS + 1];
static float m_ctStickyTrapNextShot[MAXPLAYERS + 1];
static float m_ctStickyTrapAgain[MAXPLAYERS + 1];
static int m_iStickyTrapBombsLeft[MAXPLAYERS + 1];
static float m_vStickyTrapSpot[MAXPLAYERS + 1][3];
static float m_vStickyTrapPoint[MAXPLAYERS + 1][3];

BehaviorAction CTFBotStickyTrap()
{
	BehaviorAction action = ActionsManager.Create("DefenderStickyTrap");

	action.OnStart = CTFBotStickyTrap_OnStart;
	action.Update = CTFBotStickyTrap_Update;

	return action;
}

public Action CTFBotStickyTrap_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));

	m_ctStickyTrapEnd[actor] = GetGameTime() + STICKY_TRAP_MAX_TIME;
	m_ctStickyTrapNextShot[actor] = 0.0;
	m_vStickyTrapPoint[actor] = NULL_VECTOR;

	int launcher = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	int bombs = launcher != -1 ? GetEntProp(launcher, Prop_Data, "m_iClip1") : 0;

	m_iStickyTrapBombsLeft[actor] = bombs < STICKY_TRAP_BOMBS ? bombs : STICKY_TRAP_BOMBS;

	if (!StickyTrapSpot(m_vStickyTrapSpot[actor]))
		m_iStickyTrapBombsLeft[actor] = 0;

	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_SENTRYHERE);

	return action.Continue();
}

public Action CTFBotStickyTrap_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	m_ctStickyTrapAgain[actor] = GetGameTime() + STICKY_TRAP_COOLDOWN;

	if (m_iStickyTrapBombsLeft[actor] <= 0)
		return action.Done("Trap is laid");

	if (m_ctStickyTrapEnd[actor] < GetGameTime())
		return action.Done("Took too long to lay a trap");

	int launcher = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);

	if (launcher == -1 || TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER || !HasAmmo(launcher))
		return action.Done("Nothing to lay it with");

	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);

	/* Something is shooting at the bot, so the trap stops being the job
	The bombs already down are not wasted: the detonation tick blows them the moment the fight
	walks into them, whether this action laid all eight or two */
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(true);

	if (threat != NULL_KNOWN_ENTITY)
		return action.Done("Something to fight");

	float myOrigin[3]; GetClientAbsOrigin(actor, myOrigin);
	float range = GetVectorDistance(myOrigin, m_vStickyTrapSpot[actor]);

	//Too far to aim at the ground honestly. Walk in, and give up if the walk takes the deadline
	if (range > STICKY_TRAP_MAX_RANGE)
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToPos(actor, myBot, m_vStickyTrapSpot[actor]);
		}

		m_pPath[actor].Update(myBot);

		return action.Continue();
	}

	//Standing in it. Anywhere further from the trap will do, and the path back is the way it came
	if (range < STICKY_TRAP_MIN_RANGE)
		return action.Done("Standing in my own trap");

	TF2Util_SetPlayerActiveWeapon(actor, launcher);

	//A fresh point for each bomb, near enough to the last that a giant takes the whole stack
	if (IsZeroVector(m_vStickyTrapPoint[actor]))
	{
		float spread = Feature(FEATURE_STICKY_STACK) ? STICKY_TRAP_SPREAD : STICKY_TRAP_CARPET;
		
		m_vStickyTrapPoint[actor][0] = m_vStickyTrapSpot[actor][0] + GetRandomFloat(-spread, spread);
		m_vStickyTrapPoint[actor][1] = m_vStickyTrapSpot[actor][1] + GetRandomFloat(-spread, spread);
		m_vStickyTrapPoint[actor][2] = m_vStickyTrapSpot[actor][2];
	}

	IBody myBody = myBot.GetBodyInterface();

	AimHeadTowards(myBody, m_vStickyTrapPoint[actor], CRITICAL, 1.0, _, "Laying a sticky trap");

	if (m_ctStickyTrapNextShot[actor] > GetGameTime())
		return action.Continue();

	if (!myBody.IsHeadAimingOnTarget())
		return action.Continue();

	VS_PressFireButton(actor);

	m_ctStickyTrapNextShot[actor] = GetGameTime() + GetRandomFloat(STICKY_TRAP_SHOT_GAP_MIN, STICKY_TRAP_SHOT_GAP_MAX);
	m_vStickyTrapPoint[actor] = NULL_VECTOR;
	m_iStickyTrapBombsLeft[actor]--;

	return action.Continue();
}

/* The ground worth trapping, which for a defender is wherever the bomb is

Robots escort it, so it is the one piece of ground every robot on the map is walking towards, and
the carrier stands on it while the rest of them fight around it. With no bomb in play the hatch
is the same argument with the robots not there yet */
static bool StickyTrapSpot(float spot[3])
{
	BombInfo_t bombinfo;

	if (GetBombInfo(bombinfo))
	{
		spot = bombinfo.vPosition;

		return true;
	}

	spot = GetBombHatchPosition();

	return !IsZeroVector(spot);
}

void ResetStickyTrap(int client)
{
	m_ctStickyTrapAgain[client] = 0.0;
	m_iStickyTrapBombsLeft[client] = 0;
	m_vStickyTrapSpot[client] = NULL_VECTOR;
}

bool CTFBotStickyTrap_IsPossible(int client)
{
	if (TF2_GetPlayerClass(client) != TFClass_DemoMan)
		return false;

	if (m_ctStickyTrapAgain[client] > GetGameTime())
		return false;

	int launcher = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);

	if (launcher == -1 || TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER)
		return false;

	if (!HasAmmo(launcher) || GetEntProp(launcher, Prop_Data, "m_iClip1") <= 0)
		return false;

	//There is already a trap down. Another one is the same ground covered twice
	if (GetEntProp(launcher, Prop_Send, "m_iPipebombCount") >= STICKY_TRAP_ENOUGH)
		return false;

	//A fight is not the time. Laying a trap is what a Demoman does before one
	if (CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface().GetPrimaryKnownThreat(true) != NULL_KNOWN_ENTITY)
		return false;

	float spot[3];

	if (!StickyTrapSpot(spot))
		return false;

	float myOrigin[3]; GetClientAbsOrigin(client, myOrigin);
	float range = GetVectorDistance(myOrigin, spot);

	return range > STICKY_TRAP_MIN_RANGE && range < STICKY_TRAP_MAX_RANGE;
}
