BehaviorAction CTFBotAttackTank()
{
	BehaviorAction action = ActionsManager.Create("DefenderAttackTank");

	action.OnStart = CTFBotAttackTank_OnStart;
	action.Update = CTFBotAttackTank_Update;
	action.SelectMoreDangerousThreat = CTFBotAttackTank_SelectMoreDangerousThreat;

	return action;
}

#define Go_Slots (65)

#define TANK_ATTACK_RANGE_MELEE (1.0)
#define TANK_ATTACK_RANGE_SPLASH (400.0)
#define TANK_ATTACK_RANGE_DEFAULT (100.0)

#define TANK_BLAST_SAFE_RANGE (250.0)

int m_iTankTarget[65];

public Action CTFBotAttackTank_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	return action.Continue();
}

public Action CTFBotAttackTank_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidEntity(m_iTankTarget[actor]))
	{
		if (!CTFBotAttackTank_SelectTarget(actor))
		{
			return action.Done("No valid target");
		}
	}
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Scout:
		{
			if (CTFBotCollectMoney_IsPossible(actor))
			{
				return action.ChangeTo(CTFBotCollectMoney(), "Get credits");
			}
		}
		case TFClass_Heavy, TFClass_Sniper:
		{
			if (CTFBotDefenderAttack_SelectTarget(actor))
			{
				return action.ChangeTo(CTFBotDefenderAttack(), "Robot priority");
			}
		}
	}
	EquipBestTankWeapon(actor);
	float myEyePos[3];
	GetClientEyePosition(actor, myEyePos);
	float targetOrigin[3];
	targetOrigin = WorldSpaceCenter(m_iTankTarget[actor]);
	float distToTank = GetVectorDistance(myEyePos, targetOrigin);
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	float attackRange = GetIdealTankAttackRange(actor);
	if (IsBlastWeapon(BaseCombatCharacter_GetActiveWeapon(actor)) && (RangeToTankHull(myEyePos, m_iTankTarget[actor]) < TANK_BLAST_SAFE_RANGE))
	{
		float away[3];
		SubtractVectors(myEyePos, targetOrigin, away);
		away[2] = 0.0;
		float length = NormalizeVector(away, away);
		if (length > 0.0)
		{
			ScaleVector(away, TANK_BLAST_SAFE_RANGE);
			AddVectors(myEyePos, away, away);
			myBot.GetLocomotionInterface().Approach(away, 1.0);
		}
		return action.Continue();
	}
	if ((distToTank > attackRange) || !IsLineOfFireClearEntity(actor, myEyePos, m_iTankTarget[actor]))
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.5, 1.0);
			m_pPath[actor].ComputeToPos(myBot, GetAbsOrigin(m_iTankTarget[actor]), 0.0, true);
		}
		m_pPath[actor].Update(myBot);
	}
	return action.Continue();
}

public Action CTFBotAttackTank_SelectMoreDangerousThreat(BehaviorAction action, INextBot nextbot, int entity, CKnownEntity threat1, CKnownEntity threat2, CKnownEntity& knownEntity)
{
	knownEntity = view_as<CKnownEntity>(0);
	int iThreat1 = threat1.GetEntity();
	int iThreat2 = threat2.GetEntity();
	int me = action.Actor;
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(me);
	if ((myWeapon != -1) && IsMeleeWeapon(myWeapon))
	{
		knownEntity = SelectCloserThreat(nextbot, threat1, threat2);
		return Plugin_Changed;
	}
	float notSafeRange = FLAMETHROWER_REACH_RANGE;
	if (BaseEntity_IsPlayer(iThreat1))
	{
		if (nextbot.IsRangeLessThan(iThreat1, notSafeRange))
		{
			knownEntity = threat1;
			return Plugin_Changed;
		}
	}
	if (BaseEntity_IsPlayer(iThreat2))
	{
		if (nextbot.IsRangeLessThan(iThreat2, notSafeRange))
		{
			knownEntity = threat2;
			return Plugin_Changed;
		}
	}
	if (iThreat1 == m_iTankTarget[me])
	{
		knownEntity = threat1;
		return Plugin_Changed;
	}
	if (iThreat2 == m_iTankTarget[me])
	{
		knownEntity = threat2;
		return Plugin_Changed;
	}
	knownEntity = NULL_KNOWN_ENTITY;
	return Plugin_Changed;
}

stock bool CTFBotAttackTank_SelectTarget(int actor)
{
	if (GetCountOfBotsWithNamedAction("DefenderAttackTank", actor) >= redbots_manager_bot_max_tank_attackers.IntValue)
	{
		return false;
	}
	m_iTankTarget[actor] = GetTankToTarget(actor, 999999.0);
	return m_iTankTarget[actor] != -1;
}

stock int GetTankToTarget(int actor, float maxDistance = 999999.0)
{
	float origin[3];
	GetClientAbsOrigin(actor, origin);
	int myTeam = GetClientTeam(actor);
	int primary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Primary);
	int primaryID = -1;
	if (primary != -1)
	{
		primaryID = view_as<int>(TF2Util_GetWeaponID(primary));
	}
	float bestDistance = 999999.0;
	int bestEntity = -1;
	int ent = -1;
	for (;;)
	{
		ent = FindEntityByClassname(ent, "tank_boss");
		if (ent == -1)
		{
			break;
		}
		if (myTeam == BaseEntity_GetTeamNumber(ent))
		{
			continue;
		}
		if (primaryID == view_as<int>(TF_WEAPON_FLAMETHROWER))
		{
			if ((GetEntityFlags(ent) & FL_ONGROUND) == 0)
			{
				continue;
			}
		}
		float distance = GetVectorDistance(origin, WorldSpaceCenter(ent));
		if ((distance <= bestDistance) && (distance <= maxDistance))
		{
			bestDistance = distance;
			bestEntity = ent;
		}
	}
	return bestEntity;
}

stock float GetIdealTankAttackRange(int client)
{
	int weapon = BaseCombatCharacter_GetActiveWeapon(client);
	if (weapon > 0)
	{
		if (IsMeleeWeapon(weapon))
		{
			return TANK_ATTACK_RANGE_MELEE;
		}
		switch (TF2Util_GetWeaponID(weapon))
		{
			case TF_WEAPON_ROCKETLAUNCHER, TF_WEAPON_GRENADELAUNCHER, TF_WEAPON_FLAREGUN, TF_WEAPON_DIRECTHIT, TF_WEAPON_PARTICLE_CANNON, TF_WEAPON_CANNON:
			{
				return TANK_ATTACK_RANGE_SPLASH;
			}
		}
	}
	return TANK_ATTACK_RANGE_DEFAULT;
}

stock bool IsBlastWeapon(int weapon)
{
	if (weapon < 1)
	{
		return false;
	}
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_ROCKETLAUNCHER, TF_WEAPON_GRENADELAUNCHER, TF_WEAPON_DIRECTHIT, TF_WEAPON_PARTICLE_CANNON, TF_WEAPON_CANNON, TF_WEAPON_PIPEBOMBLAUNCHER:
		{
			return true;
		}
	}
	return false;
}

stock float RangeToTankHull(float from[3], int tank)
{
	float origin[3];
	origin = GetAbsOrigin(tank);
	if (!HasEntProp(tank, Prop_Send, "m_vecMins") || !HasEntProp(tank, Prop_Send, "m_vecMaxs"))
	{
		return GetVectorDistance(from, WorldSpaceCenter(tank));
	}
	float mins[3];
	GetEntPropVector(tank, Prop_Send, "m_vecMins", mins);
	float maxs[3];
	GetEntPropVector(tank, Prop_Send, "m_vecMaxs", maxs);
	float closest[3];
	for (int axis = 0; axis < 3; axis++)
	{
		closest[axis] = ClampFloat(from[axis], origin[axis] + mins[axis], origin[axis] + maxs[axis]);
	}
	return GetVectorDistance(from, closest);
}

stock void EquipBestTankWeapon(int client)
{
	int bestWeapon = -1;
	int bestScore = 0;
	for (int slot = TFWeaponSlot_Primary; slot <= TFWeaponSlot_Melee; slot++)
	{
		int weapon = GetPlayerWeaponSlot(client, slot);
		if (weapon == -1)
		{
			continue;
		}
		int score = 0;
		switch (TF2_GetPlayerClass(client))
		{
			case TFClass_Scout:
			{
				score = EvalTankWeapon_Scout(slot, weapon);
			}
			case TFClass_Sniper:
			{
				score = EvalTankWeapon_Sniper(slot, weapon);
			}
			case TFClass_Soldier:
			{
				score = EvalTankWeapon_Soldier(slot, weapon);
			}
			case TFClass_DemoMan:
			{
				score = EvalTankWeapon_Demo(slot, weapon);
			}
			case TFClass_Medic:
			{
				score = EvalTankWeapon_Medic(slot, weapon);
			}
			case TFClass_Heavy:
			{
				score = EvalTankWeapon_Heavy(slot, weapon);
			}
			case TFClass_Pyro:
			{
				score = EvalTankWeapon_Pyro(slot, weapon);
			}
			case TFClass_Spy:
			{
				score = EvalTankWeapon_Spy(slot, weapon);
			}
			case TFClass_Engineer:
			{
				score = EvalTankWeapon_Engie(slot, weapon);
			}
		}
		if ((bestWeapon == -1) || (score > bestScore))
		{
			bestWeapon = weapon;
			bestScore = score;
		}
	}
	if (bestWeapon == -1)
	{
		LogError("EquipBestTankWeapon: no valid weapons!");
		return;
	}
	TF2Util_SetPlayerActiveWeapon(client, bestWeapon);
}

stock int EvalTankWeapon_Scout(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_SCATTERGUN, TF_WEAPON_PEP_BRAWLER_BLASTER, TF_WEAPON_SODA_POPPER, TF_WEAPON_HANDGUN_SCOUT_PRIMARY:
		{
			return 100;
		}
		case TF_WEAPON_BAT, TF_WEAPON_BAT_FISH, TF_WEAPON_BAT_WOOD:
		{
			return 80;
		}
		case TF_WEAPON_PISTOL, TF_WEAPON_PISTOL_SCOUT, TF_WEAPON_HANDGUN_SCOUT_SEC:
		{
			return 60;
		}
		case TF_WEAPON_BAT_GIFTWRAP:
		{
			return 20;
		}
		case TF_WEAPON_CLEAVER, TF_WEAPON_LUNCHBOX, TF_WEAPON_JAR, TF_WEAPON_JAR_MILK:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 60;
		}
		case TFWeaponSlot_Melee:
		{
			return 80;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Soldier(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_ROCKETLAUNCHER, TF_WEAPON_PARTICLE_CANNON, TF_WEAPON_DIRECTHIT:
		{
			return 100;
		}
		case TF_WEAPON_SHOVEL, TF_WEAPON_BOTTLE, TF_WEAPON_SWORD:
		{
			return 80;
		}
		case TF_WEAPON_SHOTGUN_PRIMARY, TF_WEAPON_SHOTGUN_SOLDIER, TF_WEAPON_SHOTGUN_HWG, TF_WEAPON_SHOTGUN_PYRO, TF_WEAPON_RAYGUN:
		{
			return 60;
		}
		case TF_WEAPON_BUFF_ITEM, TF_WEAPON_PARACHUTE:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 60;
		}
		case TFWeaponSlot_Melee:
		{
			return 80;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Pyro(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_FLAMETHROWER, TF_WEAPON_FLAMETHROWER_ROCKET:
		{
			return 100;
		}
		case TF_WEAPON_FIREAXE:
		{
			return 80;
		}
		case TF_WEAPON_SHOTGUN_PRIMARY, TF_WEAPON_SHOTGUN_SOLDIER, TF_WEAPON_SHOTGUN_HWG, TF_WEAPON_SHOTGUN_PYRO:
		{
			return 60;
		}
		case TF_WEAPON_FLAREGUN, TF_WEAPON_RAYGUN_REVENGE:
		{
			return 20;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 20;
		}
		case TFWeaponSlot_Melee:
		{
			return 80;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Demo(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_GRENADELAUNCHER, TF_WEAPON_CANNON:
		{
			return 100;
		}
		case TF_WEAPON_PIPEBOMBLAUNCHER:
		{
			if (Feature(FEATURE_DEMO_TANK_PIPES))
			{
				return 0;
			}
			return 110;
		}
		case TF_WEAPON_BOTTLE, TF_WEAPON_SHOVEL, TF_WEAPON_SWORD, TF_WEAPON_STICKBOMB:
		{
			return 80;
		}
		case TF_WEAPON_BUFF_ITEM, TF_WEAPON_PARACHUTE, TF_WEAPON_STICKY_BALL_LAUNCHER:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 0;
		}
		case TFWeaponSlot_Melee:
		{
			return 80;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Heavy(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_MINIGUN:
		{
			return 100;
		}
		case TF_WEAPON_FISTS, TF_WEAPON_FIREAXE:
		{
			return 80;
		}
		case TF_WEAPON_SHOTGUN_PRIMARY, TF_WEAPON_SHOTGUN_SOLDIER, TF_WEAPON_SHOTGUN_HWG, TF_WEAPON_SHOTGUN_PYRO:
		{
			return 60;
		}
		case TF_WEAPON_LUNCHBOX:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 0;
		}
		case TFWeaponSlot_Melee:
		{
			return 60;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Engie(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_WRENCH, TF_WEAPON_MECHANICAL_ARM:
		{
			return 100;
		}
		case TF_WEAPON_SHOTGUN_PRIMARY, TF_WEAPON_SHOTGUN_SOLDIER, TF_WEAPON_SHOTGUN_HWG, TF_WEAPON_SHOTGUN_PYRO, TF_WEAPON_SENTRY_REVENGE, TF_WEAPON_DRG_POMSON:
		{
			return 80;
		}
		case TF_WEAPON_SHOTGUN_BUILDING_RESCUE:
		{
			return 60;
		}
		case TF_WEAPON_PISTOL, TF_WEAPON_PISTOL_SCOUT, TF_WEAPON_REVOLVER:
		{
			return 40;
		}
		case TF_WEAPON_PDA, TF_WEAPON_PDA_ENGINEER_BUILD, TF_WEAPON_PDA_ENGINEER_DESTROY, TF_WEAPON_PDA_SPY, TF_WEAPON_PDA_SPY_BUILD, TF_WEAPON_BUILDER, TF_WEAPON_LASER_POINTER, TF_WEAPON_DISPENSER, TF_WEAPON_DISPENSER_GUN:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 80;
		}
		case TFWeaponSlot_Secondary:
		{
			return 0;
		}
		case TFWeaponSlot_Melee:
		{
			return 100;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Medic(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_BONESAW, TF_WEAPON_HARVESTER_SAW:
		{
			return 100;
		}
		case TF_WEAPON_SYRINGEGUN_MEDIC, TF_WEAPON_NAILGUN:
		{
			return 80;
		}
		case TF_WEAPON_CROSSBOW:
		{
			return 60;
		}
		case TF_WEAPON_MEDIGUN:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 80;
		}
		case TFWeaponSlot_Secondary:
		{
			return 0;
		}
		case TFWeaponSlot_Melee:
		{
			return 100;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Sniper(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_SNIPERRIFLE, TF_WEAPON_SNIPERRIFLE_DECAP, TF_WEAPON_SNIPERRIFLE_CLASSIC:
		{
			return 100;
		}
		case TF_WEAPON_COMPOUND_BOW:
		{
			return 80;
		}
		case TF_WEAPON_CLUB:
		{
			return 60;
		}
		case TF_WEAPON_CHARGED_SMG, TF_WEAPON_SMG:
		{
			return 40;
		}
		case TF_WEAPON_JAR, TF_WEAPON_JAR_MILK:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 40;
		}
		case TFWeaponSlot_Melee:
		{
			return 60;
		}
		default:
		{
			return 10;
		}
	}
}

stock int EvalTankWeapon_Spy(int slot, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_REVOLVER:
		{
			return 100;
		}
		case TF_WEAPON_KNIFE:
		{
			return 80;
		}
		case TF_WEAPON_PDA, TF_WEAPON_PDA_ENGINEER_BUILD, TF_WEAPON_PDA_ENGINEER_DESTROY, TF_WEAPON_PDA_SPY, TF_WEAPON_PDA_SPY_BUILD, TF_WEAPON_BUILDER, TF_WEAPON_INVIS:
		{
			return 0;
		}
	}
	switch (slot)
	{
		case TFWeaponSlot_Primary:
		{
			return 100;
		}
		case TFWeaponSlot_Secondary:
		{
			return 0;
		}
		case TFWeaponSlot_Melee:
		{
			return 80;
		}
		default:
		{
			return 10;
		}
	}
}

stock void Go_ResetAttackTank(int client)
{
	m_iTankTarget[client] = -1;
}

