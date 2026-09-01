#define MEDIC_REVIVE_RANGE	600.0

BehaviorAction CTFBotMedicRevive()
{
	BehaviorAction action = ActionsManager.Create("DefenderMedicRevive");
	
	action.OnStart = CTFBotMedicRevive_OnStart;
	action.Update = CTFBotMedicRevive_Update;
	action.OnInjured = CTFBotMedicRevive_OnInjured;
	
	return action;
}

public Action CTFBotMedicRevive_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	
	return action.Continue();
}

public Action CTFBotMedicRevive_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	
	if (secondary == -1)
		return action.Done("No medigun!");
	
	int marker = GetNearestReviveMarker(actor, MEDIC_REVIVE_RANGE);
	
	if (marker == -1)
		return action.Done("No reanimator!");
	
	float markerPos[3]; markerPos = WorldSpaceCenter(marker);
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	
	if (myBot.IsRangeLessThanEx(markerPos, WEAPON_MEDIGUN_RANGE))
	{
		int healTarget = GetEntPropEnt(secondary, Prop_Send, "m_hHealingTarget");
		
		if (healTarget != -1 && healTarget != marker)
		{
			//We're healing something that's not the revive marker, stop holding the attack button
		}
		else
		{
			TF2Util_SetPlayerActiveWeapon(actor, secondary);
			SnapViewToPosition(actor, markerPos);
			VS_PressFireButton(actor);
		}
		
		//Do not path if we are healing our target
		if (healTarget == marker)
			return action.Continue();
	}
	else
	{
		//Fend off from enemies
		int primary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Primary);
		
		if (primary != -1)
			TF2Util_SetPlayerActiveWeapon(actor, primary);
	}
	
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.9, 1.2);
		RepathToPos(actor, myBot, markerPos);
	}
	
	m_pPath[actor].Update(myBot);
	
	return action.Continue();
}

public Action CTFBotMedicRevive_OnInjured(BehaviorAction action, int actor, Address takedamageinfo, ActionDesiredResult result)
{
	CTakeDamageInfo info = CTakeDamageInfo(takedamageinfo);
	
	if (info.GetDamage() > 0.0)
	{
		int weapon = BaseCombatCharacter_GetActiveWeapon(actor);
		
		//Someone hit me while I'm trying to revive someone, let's pop uber now if possible
		if (weapon != -1 && TF2Util_GetWeaponID(weapon) == TF_WEAPON_MEDIGUN)
			VS_PressAltFireButton(actor);
	}
	
	return action.Continue();
}

/* Whether there is somebody to revive, held for a moment after it is worked out

The reachability test is a full NavAreaBuildPath and this runs on the tactical monitor's frame,
twice: once from the game's heal action and once from the mod's. A wave leaves revive markers
lying about wherever a defender died, so a medic in a fight had one in range most of the time and
paid for a nav mesh search on every frame of it to be told what it was told last frame.

That is the third call of this shape found in one night, after the health and ammo search and the
nest scoring. The pattern is worth naming: a nav mesh question inside something that reads like a
cheap predicate, on a path that runs every frame.

Half a second, which is a medic walking about a hundred and fifty units. A marker does not appear
and expire inside that. */
#define MEDIC_REVIVE_ASK_INTERVAL	0.5

static float m_ctReviveAsk[MAXPLAYERS + 1];
static bool m_bRevivePossible[MAXPLAYERS + 1];

bool CTFBotMedicRevive_IsPossible(int client)
{
	if (m_ctReviveAsk[client] > GetGameTime())
		return m_bRevivePossible[client];
	
	m_ctReviveAsk[client] = GetGameTime() + MEDIC_REVIVE_ASK_INTERVAL;
	
	m_bRevivePossible[client] = false;
	
	int marker = GetNearestReviveMarker(client, MEDIC_REVIVE_RANGE);
	
	if (marker == -1)
		return false;
	
	if (!IsPathToVectorPossible(client, GetAbsOrigin(marker)))
		return false;
	
	m_bRevivePossible[client] = true;
	
	return true;
}