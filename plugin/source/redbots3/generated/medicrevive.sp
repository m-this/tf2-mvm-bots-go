BehaviorAction CTFBotMedicRevive()
{
	BehaviorAction action = ActionsManager.Create("DefenderMedicRevive");

	action.OnStart = CTFBotMedicRevive_OnStart;
	action.Update = CTFBotMedicRevive_Update;
	action.OnInjured = CTFBotMedicRevive_OnInjured;

	return action;
}

#define MEDIC_REVIVE_RANGE (600.0)

#define MEDIC_REVIVE_ASK_INTERVAL (0.5)

float m_ctReviveAsk[65];
bool m_bRevivePossible[65];

// OnStart aims the path.
public Action CTFBotMedicRevive_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	return action.Continue();
}

// Update holds the beam on the marker, and pulls the primary out on the way
// there so the medic is not defenceless.
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
		// The empty branch is the shipped file's, and it is the point: a
		// medic already beaming something else stops pressing fire, and
		// writing that as a negated condition would read as a different
		// decision to anybody diffing the two.
		if ((healTarget != -1) && (healTarget != marker))
		{
		}
		else
		{
			TF2Util_SetPlayerActiveWeapon(actor, secondary);
			SnapViewToPosition(actor, markerPos);
			VS_PressFireButton(actor);
		}
		// Do not path if we are healing our target
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

// OnInjured pops the uber when something hits the medic mid-revive.
public Action CTFBotMedicRevive_OnInjured(BehaviorAction action, int actor, Address takedamageinfo, ActionDesiredResult result)
{
	CTakeDamageInfo info = CTakeDamageInfo(takedamageinfo);
	if (info.GetDamage() > 0.0)
	{
		int weapon = BaseCombatCharacter_GetActiveWeapon(actor);
		// Someone hit me while I'm trying to revive someone, let's pop uber now if possible
		if ((weapon != -1) && (TF2Util_GetWeaponID(weapon) == TF_WEAPON_MEDIGUN))
		{
			VS_PressAltFireButton(actor);
		}
	}
	return action.Continue();
}

// ResetMedicRevive forgets a seat's answer, which was about somebody else.
//
// IsPossible holds a marker within reviveRange of where the medic stood, and the
// path to it, for half a second. A bot spawning into the seat is nowhere near
// either, so it walks to a marker it cannot reach or refuses one it can.
stock void Go_ResetMedicRevive(int client)
{
	m_ctReviveAsk[client] = 0.0;
	m_bRevivePossible[client] = false;
}

// IsPossible answers whether there is anybody to revive, held for askInterval.
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

