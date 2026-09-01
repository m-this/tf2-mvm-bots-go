BehaviorAction CTFBotStickyTrap()
{
	BehaviorAction action = ActionsManager.Create("DefenderStickyTrap");

	action.OnStart = CTFBotStickyTrap_OnStart;
	action.Update = CTFBotStickyTrap_Update;

	return action;
}

#define Go_Slots (65)

#define STICKY_TRAP_BOMBS (8)

#define STICKY_TRAP_SPREAD (40.0)

#define STICKY_TRAP_CARPET (120.0)

#define STICKY_TRAP_SHOT_GAP_MIN (0.6)
#define STICKY_TRAP_SHOT_GAP_MAX (0.9)

#define STICKY_TRAP_MAX_TIME (12.0)

#define STICKY_TRAP_MIN_RANGE (350.0)
#define STICKY_TRAP_MAX_RANGE (1500.0)

#define STICKY_TRAP_ENOUGH (4)

#define STICKY_TRAP_COOLDOWN (20.0)

float m_ctStickyTrapEnd[65];
float m_ctStickyTrapNextShot[65];
float m_ctStickyTrapAgain[65];
int m_iStickyTrapBombsLeft[65];
float m_vStickyTrapSpot[65][3];
float m_vStickyTrapPoint[65][3];

public Action CTFBotStickyTrap_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_ctStickyTrapEnd[actor] = GetGameTime() + STICKY_TRAP_MAX_TIME;
	m_ctStickyTrapNextShot[actor] = 0.0;
	m_vStickyTrapPoint[actor] = NULL_VECTOR;
	int launcher = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	int bombs = 0;
	if (launcher != -1)
	{
		bombs = GetEntProp(launcher, Prop_Data, "m_iClip1");
	}
	m_iStickyTrapBombsLeft[actor] = STICKY_TRAP_BOMBS;
	if (bombs < STICKY_TRAP_BOMBS)
	{
		m_iStickyTrapBombsLeft[actor] = bombs;
	}
	float spot[3];
	bool found = StickyTrapSpot(spot);
	m_vStickyTrapSpot[actor] = spot;
	if (!found)
	{
		m_iStickyTrapBombsLeft[actor] = 0;
	}
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_SENTRYHERE);
	return action.Continue();
}

public Action CTFBotStickyTrap_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	m_ctStickyTrapAgain[actor] = GetGameTime() + STICKY_TRAP_COOLDOWN;
	if (m_iStickyTrapBombsLeft[actor] <= 0)
	{
		return action.Done("Trap is laid");
	}
	if (m_ctStickyTrapEnd[actor] < GetGameTime())
	{
		return action.Done("Took too long to lay a trap");
	}
	int launcher = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	if ((launcher == -1) || (TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER) || !HasAmmo(launcher))
	{
		return action.Done("Nothing to lay it with");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(true);
	if (threat != NULL_KNOWN_ENTITY)
	{
		return action.Done("Something to fight");
	}
	float myOrigin[3];
	GetClientAbsOrigin(actor, myOrigin);
	float trapRange = GetVectorDistance(myOrigin, m_vStickyTrapSpot[actor]);
	if (trapRange > STICKY_TRAP_MAX_RANGE)
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToPos(actor, myBot, m_vStickyTrapSpot[actor]);
		}
		m_pPath[actor].Update(myBot);
		return action.Continue();
	}
	if (trapRange < STICKY_TRAP_MIN_RANGE)
	{
		return action.Done("Standing in my own trap");
	}
	TF2Util_SetPlayerActiveWeapon(actor, launcher);
	if (IsZeroVector(m_vStickyTrapPoint[actor]))
	{
		float spread = 120.0;
		if (Feature(FEATURE_STICKY_STACK))
		{
			spread = STICKY_TRAP_SPREAD;
		}
		m_vStickyTrapPoint[actor][0] = m_vStickyTrapSpot[actor][0] + GetRandomFloat(-spread, spread);
		m_vStickyTrapPoint[actor][1] = m_vStickyTrapSpot[actor][1] + GetRandomFloat(-spread, spread);
		m_vStickyTrapPoint[actor][2] = m_vStickyTrapSpot[actor][2];
	}
	IBody myBody = myBot.GetBodyInterface();
	AimHeadTowards(myBody, m_vStickyTrapPoint[actor], CRITICAL, 1.0, Address_Null, "Laying a sticky trap");
	if (m_ctStickyTrapNextShot[actor] > GetGameTime())
	{
		return action.Continue();
	}
	if (!myBody.IsHeadAimingOnTarget())
	{
		return action.Continue();
	}
	VS_PressFireButton(actor);
	m_ctStickyTrapNextShot[actor] = GetGameTime() + GetRandomFloat(STICKY_TRAP_SHOT_GAP_MIN, STICKY_TRAP_SHOT_GAP_MAX);
	m_vStickyTrapPoint[actor] = NULL_VECTOR;
	m_iStickyTrapBombsLeft[actor]--;
	return action.Continue();
}

stock bool StickyTrapSpot(float spot[3])
{
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	BombInfo_t bombinfo;
	bool haveBomb = GetBombInfo(bombinfo);
	if (haveBomb)
	{
		spot = bombinfo.vPosition;
		return true;
	}
	spot = GetBombHatchPosition();
	return !IsZeroVector(spot);
}

stock void ResetStickyTrap(int client)
{
	m_ctStickyTrapAgain[client] = 0.0;
	m_iStickyTrapBombsLeft[client] = 0;
	m_vStickyTrapSpot[client] = NULL_VECTOR;
}

stock bool CTFBotStickyTrap_IsPossible(int client)
{
	if (TF2_GetPlayerClass(client) != TFClass_DemoMan)
	{
		return false;
	}
	if (m_ctStickyTrapAgain[client] > GetGameTime())
	{
		return false;
	}
	int launcher = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
	if ((launcher == -1) || (TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER))
	{
		return false;
	}
	if (!HasAmmo(launcher) || (GetEntProp(launcher, Prop_Data, "m_iClip1") <= 0))
	{
		return false;
	}
	if (GetEntProp(launcher, Prop_Send, "m_iPipebombCount") >= STICKY_TRAP_ENOUGH)
	{
		return false;
	}
	if (CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface().GetPrimaryKnownThreat(true) != NULL_KNOWN_ENTITY)
	{
		return false;
	}
	float spot[3];
	bool found = StickyTrapSpot(spot);
	if (!found)
	{
		return false;
	}
	float myOrigin[3];
	GetClientAbsOrigin(client, myOrigin);
	float trapRange = GetVectorDistance(myOrigin, spot);
	return (trapRange > STICKY_TRAP_MIN_RANGE) && (trapRange < STICKY_TRAP_MAX_RANGE);
}

