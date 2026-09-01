BehaviorAction CTFBotMedicRevive()
{
	BehaviorAction action = ActionsManager.Create("DefenderMedicRevive");

	action.OnStart = CTFBotMedicRevive_OnStart;
	action.Update = CTFBotMedicRevive_Update;
	action.OnInjured = CTFBotMedicRevive_OnInjured;

	return action;
}

#define Go_Slots (65)

#define MEDIC_REVIVE_RANGE (600.0)

#define MEDIC_REVIVE_ASK_INTERVAL (0.5)

float m_ctReviveAsk[65];
bool m_bRevivePossible[65];

public Action CTFBotMedicRevive_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	return action.Continue();
}

public Action CTFBotMedicRevive_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	if (secondary == -1)
	{
		return action.Done("No medigun!");
	}
	int marker = GetNearestReviveMarker(actor, MEDIC_REVIVE_RANGE);
	if (marker == -1)
	{
		return action.Done("No reanimator!");
	}
	float markerPos[3];
	markerPos = WorldSpaceCenter(marker);
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (myBot.IsRangeLessThanEx(markerPos, WEAPON_MEDIGUN_RANGE))
	{
		int healTarget = GetEntPropEnt(secondary, Prop_Send, "m_hHealingTarget");
		if ((healTarget != -1) && (healTarget != marker))
		{
		}
		else
		{
			TF2Util_SetPlayerActiveWeapon(actor, secondary);
			SnapViewToPosition(actor, markerPos);
			VS_PressFireButton(actor);
		}
		if (healTarget == marker)
		{
			return action.Continue();
		}
	}
	else
	{
		int primary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Primary);
		if (primary != -1)
		{
			TF2Util_SetPlayerActiveWeapon(actor, primary);
		}
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
		if ((weapon != -1) && (TF2Util_GetWeaponID(weapon) == TF_WEAPON_MEDIGUN))
		{
			VS_PressAltFireButton(actor);
		}
	}
	return action.Continue();
}

stock bool CTFBotMedicRevive_IsPossible(int client)
{
	if (m_ctReviveAsk[client] > GetGameTime())
	{
		return m_bRevivePossible[client];
	}
	m_ctReviveAsk[client] = GetGameTime() + MEDIC_REVIVE_ASK_INTERVAL;
	m_bRevivePossible[client] = false;
	int marker = GetNearestReviveMarker(client, MEDIC_REVIVE_RANGE);
	if (marker == -1)
	{
		return false;
	}
	if (!IsPathToVectorPossible(client, GetAbsOrigin(marker)))
	{
		return false;
	}
	m_bRevivePossible[client] = true;
	return true;
}

