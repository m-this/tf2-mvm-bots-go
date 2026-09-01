BehaviorAction CTFBotGetHealth()
{
	BehaviorAction action = ActionsManager.Create("DefenderGetHealth");

	action.OnStart = CTFBotGetHealth_OnStart;
	action.Update = CTFBotGetHealth_Update;
	action.OnEnd = CTFBotGetHealth_OnEnd;
	action.ShouldHurry = CTFBotGetHealth_ShouldHurry;
	action.ShouldAttack = CTFBotGetHealth_ShouldAttack;

	return action;
}

#define Go_Slots (65)

#define HEALTH_ASK_INTERVAL (0.5)

int m_iHealthPack[65];
float m_ctHealthAsk[65];
bool m_bHealthPossible[65];

public Action CTFBotGetHealth_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	float healthRatio = float(GetClientHealth(actor)) / float(TEMP_GetPlayerMaxHealth(actor));
	float ratio = ClampFloat((healthRatio - tf_bot_health_critical_ratio.FloatValue) / (tf_bot_health_ok_ratio.FloatValue - tf_bot_health_critical_ratio.FloatValue), 0.0, 1.0);
	float farRange = tf_bot_health_search_far_range.FloatValue;
	float maxRange = ratio * (tf_bot_health_search_near_range.FloatValue - farRange);
	maxRange += farRange;
	ArrayList ammo = new ArrayList(2);
	ComputeHealthAndAmmoVectors(actor, ammo, maxRange);
	if (ammo.Length <= 0)
	{
		delete ammo;
		return action.Done("No health");
	}
	float flSmallestDistance = 99999.0;
	for (int i = 0; i < ammo.Length; i++)
	{
		int entity = ammo.Get(i, 0);
		if (!IsValidHealth(entity))
		{
			continue;
		}
		float flDistance = view_as<float>(ammo.Get(i, 1));
		if (flDistance <= flSmallestDistance)
		{
			m_iHealthPack[actor] = entity;
			flSmallestDistance = flDistance;
		}
	}
	if (m_iHealthPack[actor] != -1)
	{
		if (TF2_GetPlayerClass(actor) == TFClass_Engineer)
		{
			UpdateLookAroundForEnemies(actor, true);
		}
		BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_MEDIC);
		delete ammo;
		return action.Continue();
	}
	delete ammo;
	return action.Done("Could not find health");
}

public Action CTFBotGetHealth_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidHealth(m_iHealthPack[actor]))
	{
		return action.Done("Health is not valid");
	}
	if (IsHealedByMedic(actor))
	{
		return action.Done("A medic heals me");
	}
	if (GetClientHealth(actor) >= TEMP_GetPlayerMaxHealth(actor))
	{
		return action.Done("I am healed");
	}
	if (TF2_IsCarryingObject(actor))
	{
		VS_PressFireButton(actor);
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (IsHealedByObject(actor))
	{
		int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
		if ((myWeapon != -1) && WeaponID_IsSniperRifle(TF2Util_GetWeaponID(myWeapon)) && !TF2_IsPlayerInCondition(actor, TFCond_Zoomed))
		{
			VS_PressAltFireButton(actor);
		}
	}
	else
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.9, 1.0);
			RepathToPos(actor, myBot, WorldSpaceCenter(m_iHealthPack[actor]));
		}
		m_pPath[actor].Update(myBot);
	}
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		EquipBestWeaponForThreat(actor, threat);
	}
	return action.Continue();
}

public void CTFBotGetHealth_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iHealthPack[actor] = -1;
}

public Action CTFBotGetHealth_ShouldHurry(BehaviorAction action, INextBot nextbot, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	result = ANSWER_YES;
	return Plugin_Changed;
}

public Action CTFBotGetHealth_ShouldAttack(BehaviorAction action, INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	int me = action.Actor;
	if (TF2_GetPlayerClass(me) == TFClass_Spy)
	{
		int iThreat = knownEntity.GetEntity();
		if (BaseEntity_IsPlayer(iThreat) && (GetClientHealth(iThreat) > 360) && !TF2_IsCritBoosted(me))
		{
			result = ANSWER_NO;
			return Plugin_Changed;
		}
		else
			if (GetNearestEnemyCount(me, 1000.0, false) > 1)
			{
				result = ANSWER_NO;
				return Plugin_Changed;
			}
	}
	result = ANSWER_UNDEFINED;
	return Plugin_Changed;
}

stock bool IsValidHealth(int pack)
{
	if (!IsValidEntity(pack))
	{
		return false;
	}
	if (!HasEntProp(pack, Prop_Send, "m_fEffects"))
	{
		return false;
	}
	if (GetEntProp(pack, Prop_Send, "m_fEffects") != 0)
	{
		return false;
	}
	char class[512];
	GetEntityClassname(pack, class, 512);
	if ((StrContains(class, "item_health", false) == -1) && (StrContains(class, "obj_dispenser", false) == -1) && (StrContains(class, "func_regen", false) == -1))
	{
		return false;
	}
	if ((StrContains(class, "obj_dispenser", false) != -1) && TF2_HasSapper(pack))
	{
		return false;
	}
	return true;
}

stock bool CTFBotGetHealth_IsPossible(int actor)
{
	if (IsHealedByMedic(actor) || TF2_IsInvulnerable(actor))
	{
		return false;
	}
	float healthRatio = float(GetClientHealth(actor)) / float(TEMP_GetPlayerMaxHealth(actor));
	float ratio = ClampFloat((healthRatio - tf_bot_health_critical_ratio.FloatValue) / (tf_bot_health_ok_ratio.FloatValue - tf_bot_health_critical_ratio.FloatValue), 0.0, 1.0);
	float farRange = tf_bot_health_search_far_range.FloatValue;
	float maxRange = ratio * (tf_bot_health_search_near_range.FloatValue - farRange);
	maxRange += farRange;
	if ((m_iHealthPack[actor] != -1) && IsValidHealth(m_iHealthPack[actor]))
	{
		return true;
	}
	if (m_ctHealthAsk[actor] > GetGameTime())
	{
		return m_bHealthPossible[actor];
	}
	m_ctHealthAsk[actor] = GetGameTime() + HEALTH_ASK_INTERVAL;
	if (redbots_manager_debug_actions.BoolValue)
	{
		PrintToServer("ratio %f max_range %f", ratio, maxRange);
	}
	ArrayList ammo = new ArrayList(2);
	ComputeHealthAndAmmoVectors(actor, ammo, maxRange);
	bool bPossible = false;
	for (int i = 0; i < ammo.Length; i++)
	{
		if (!IsValidHealth(ammo.Get(i, 0)))
		{
			continue;
		}
		bPossible = true;
		break;
	}
	m_bHealthPossible[actor] = bPossible;
	delete ammo;
	return bPossible;
}

stock void Go_ResetGetHealth(int client)
{
	m_iHealthPack[client] = -1;
}

