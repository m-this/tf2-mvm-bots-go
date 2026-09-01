BehaviorAction CTFBotCampBomb()
{
	BehaviorAction action = ActionsManager.Create("DefenderCampBomb");

	action.OnStart = CTFBotCampBomb_OnStart;
	action.Update = CTFBotCampBomb_Update;

	return action;
}

#define BOMB_HATCH_RANGE_OKAY (5000.0)
#define BOMB_HATCH_RANGE_CRITICAL (1000.0)
#define BOMB_GUARD_RADIUS (400.0)

#define Go_maxWatchRadius (1000.0)

public Action CTFBotCampBomb_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_SENTRYHERE);
	return action.Continue();
}

public Action CTFBotCampBomb_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Soldier, TFClass_Pyro, TFClass_DemoMan:
		{
			if (CTFBotAttackTank_SelectTarget(actor))
			{
				return action.ChangeTo(CTFBotAttackTank(), "Tank inbound");
			}
		}
	}
	int flag = FindBombNearestToHatch();
	if (flag == -1)
	{
		return action.Done("No bomb");
	}
	if (BaseEntity_GetOwnerEntity(flag) != -1)
	{
		return action.ChangeTo(CTFBotDefenderAttack(), "Bomb is taken");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	float bombPosition[3];
	bombPosition = WorldSpaceCenter(flag);
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	if ((myWeapon != -1) && ((TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_FLAMETHROWER) || IsMeleeWeapon(myWeapon)))
	{
		int nearest = GetEnemyPlayerNearestToPosition(actor, bombPosition, BOMB_GUARD_RADIUS);
		if (nearest != -1)
		{
			if (m_flRepathTime[actor] <= GetGameTime())
			{
				m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.5, 1.0);
				RepathToTarget(actor, myBot, nearest);
			}
			m_pPath[actor].Update(myBot);
			return action.Continue();
		}
	}
	float guardPosition[3];
	guardPosition = bombPosition;
	if (Feature(FEATURE_DISPENSER_GUARD) && WantsDispenser(actor))
	{
		int dispenser = FindFriendlyDispenserNear(actor, bombPosition);
		if (dispenser != -1)
		{
			guardPosition = GetAbsOrigin(dispenser);
		}
	}
	if (myBot.IsRangeGreaterThanEx(guardPosition, BOMB_GUARD_RADIUS) || !IsLineOfFireClearPosition(actor, GetEyePosition(actor), bombPosition))
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
			RepathToPos(actor, myBot, guardPosition);
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

stock bool CTFBotCampBomb_IsPossible(int client)
{
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Scout, TFClass_Medic:
		{
			return false;
		}
	}
	int flag = FindBombNearestToHatch();
	if (flag == -1)
	{
		return false;
	}
	if (BaseEntity_GetOwnerEntity(flag) != -1)
	{
		return false;
	}
	float bombPosition[3];
	bombPosition = WorldSpaceCenter(flag);
	int iEnt = -1;
	for (;;)
	{
		iEnt = FindEntityByClassname(iEnt, "obj_sentrygun");
		if (iEnt == -1)
		{
			break;
		}
		if (BaseEntity_GetTeamNumber(iEnt) != GetClientTeam(client))
		{
			continue;
		}
		if (GetVectorDistance(bombPosition, WorldSpaceCenter(iEnt)) <= Go_maxWatchRadius)
		{
			return false;
		}
	}
	if (GetCountOfBotsWithNamedAction("DefenderCampBomb") > 0)
	{
		return false;
	}
	return true;
}

