BehaviorAction CTFBotSpySapPlayers()
{
	BehaviorAction action = ActionsManager.Create("DefenderSpySapPlayer");

	action.OnStart = CTFBotSpySapPlayers_OnStart;
	action.Update = CTFBotSpySapPlayers_Update;
	action.ShouldAttack = CTFBotSpySapPlayers_ShouldAttack;
	action.IsHindrance = CTFBotSpySapPlayers_IsHindrance;

	return action;
}

#define Go_groupRadius (800.0)

int m_iPlayerSapTarget[65];

// OnStart aims the path.
public Action CTFBotSpySapPlayers_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	return action.Continue();
}

// Update walks to the target and saps it once it is close enough.
public Action CTFBotSpySapPlayers_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidClientIndex(m_iPlayerSapTarget[actor]) || !IsPlayerAlive(m_iPlayerSapTarget[actor]) || (TF2_GetClientTeam(m_iPlayerSapTarget[actor]) != GetPlayerEnemyTeam(actor)) || !IsPlayerSappable(m_iPlayerSapTarget[actor]))
	{
		return action.Done("No player to sap");
	}
	int mySapper = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	if (mySapper == -1)
	{
		return action.Done("No sapper");
	}
	TF2Util_SetPlayerActiveWeapon(actor, mySapper);
	if (TF2_IsStealthed(actor) || TF2_IsFeignDeathReady(actor))
	{
		// Can't use place a sapper while cloaked, uncloak
		VS_PressAltFireButton(actor);
	}
	else
	{
		float origin[3];
		GetClientAbsOrigin(m_iPlayerSapTarget[actor], origin);
		float myOrigin[3];
		GetClientAbsOrigin(actor, myOrigin);
		SubtractVectors(origin, myOrigin, origin);
		// If we're close enough, build a sapper on them
		if ((GetVectorLength(origin) <= SAPPER_PLAYER_BUILD_ON_RANGE) && TF2Util_CanWeaponAttack(mySapper))
		{
			BuildSapperOnEntity(actor, m_iPlayerSapTarget[actor], mySapper);
			return action.Done("Sapped player");
		}
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
		RepathToTarget(actor, myBot, m_iPlayerSapTarget[actor]);
	}
	m_pPath[actor].Update(myBot);
	return action.Continue();
}

// ShouldAttack says no: the spy is placing a sapper, not fighting.
public Action CTFBotSpySapPlayers_ShouldAttack(BehaviorAction action, INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	result = ANSWER_NO;
	return Plugin_Changed;
}

// IsHindrance avoids no one.
public Action CTFBotSpySapPlayers_IsHindrance(BehaviorAction action, INextBot nextbot, int entity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	result = ANSWER_NO;
	return Plugin_Changed;
}

// SelectTarget picks who to sap: a fast giant, then a medic with a beam out,
// then anybody in a crowd if the sapper is the robo one.
stock bool CTFBotSpySapPlayers_SelectTarget(int actor)
{
	if (!CanBuildSapper(actor))
	{
		return false;
	}
	// Get the nearest fast giant
	m_iPlayerSapTarget[actor] = GetNearestSappablePlayer(actor, 1000.0, true, TFClass_Unknown, 230.0);
	// Get the nearest medic that is healing someone
	if (m_iPlayerSapTarget[actor] == -1)
	{
		m_iPlayerSapTarget[actor] = GetNearestSappablePlayerHealingSomeone(actor, 1000.0, false, TFClass_Medic, 0.0);
	}
	if (m_iPlayerSapTarget[actor] == -1)
	{
		int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
		if ((secondary != -1) && (TF2Util_GetWeaponID(secondary) == TF_WEAPON_BUILDER) && (TF2Attrib_GetByName(secondary, "robo sapper") != Address_Null))
		{
			// If there's a group of enemies near us, let's put a sapper on one of them
			if (GetNearestEnemyCount(actor, Go_groupRadius, false) >= 4)
			{
				m_iPlayerSapTarget[actor] = GetFarthestSappablePlayer(actor, Go_groupRadius, false, TFClass_Unknown, 0.0);
			}
		}
	}
	return m_iPlayerSapTarget[actor] != -1;
}

// CanBuildSapper is CTFPlayer::CanBuild, only for the ammo the builder spends.
stock bool CanBuildSapper(int client)
{
	// Like CTFPlayer::CanBuild, only if we have ammo of TF_AMMO_GRENADES2
	return BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_GRENADES2) > 0;
}

// BuildSapperOnEntity puts one on and starts the recharge.
stock void BuildSapperOnEntity(int client, int entity, int weapon)
{
	SpawnSapper(client, entity, weapon);
	// CTFWeaponBuilder uses ammo index TF_AMMO_GRENADES2 for its effect bar
	BaseCombatCharacter_RemoveAmmo(client, 1, TF_AMMO_GRENADES2);
	StartBuilderEffectBarRegen(weapon);
}

// StartBuilderEffectBarRegen sets when the game hands the ammo back.
stock void StartBuilderEffectBarRegen(int weapon)
{
	// When recharged, game will give us ammo TF_AMMO_GRENADES2 for the sapper
	SetEntPropFloat(weapon, Prop_Send, "m_flEffectBarRegenTime", GetGameTime() + SAPPER_RECHARGE_TIME);
}

// ResetSpySapPlayer forgets the player's building this spy was sapping.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetSpySapPlayer(int client)
{
	m_iPlayerSapTarget[client] = -1;
}

