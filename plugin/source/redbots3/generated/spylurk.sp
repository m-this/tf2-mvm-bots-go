BehaviorAction CTFBotSpyLurkMvM()
{
	BehaviorAction action = ActionsManager.Create("DefenderSpyLurk");

	action.OnStart = CTFBotSpyLurkMvM_OnStart;
	action.Update = CTFBotSpyLurkMvM_Update;
	action.ShouldAttack = CTFBotSpyLurkMvM_ShouldAttack;
	action.IsHindrance = CTFBotSpyLurkMvM_IsHindrance;

	return action;
}

#define Go_behindTolerance (0.0)

#define Go_circleStrafeRange (250.0)

static Action CTFBotSpyLurkMvM_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_pChasePath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_iAttackTarget[actor] = -1;
	return action.Continue();
}

static Action CTFBotSpyLurkMvM_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (CTFBotSpySapPlayers_SelectTarget(actor))
	{
		return action.SuspendFor(CTFBotSpySapPlayers(), "Sapping player");
	}
	if (CTFBotSpySap_SelectTarget(actor))
	{
		return action.SuspendFor(CTFBotSpySap(), "Sapping building");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	int target = GetBestTargetForSpy(actor, 2000.0);
	if (target != -1)
	{
		if (TF2_IsStealthed(actor) || TF2_IsFeignDeathReady(actor))
		{
			VS_PressAltFireButton(actor);
		}
		int melee = GetPlayerWeaponSlot(actor, TFWeaponSlot_Melee);
		if (melee != -1)
		{
			TF2Util_SetPlayerActiveWeapon(actor, melee);
		}
		float playerThreatForward[3];
		BasePlayer_EyeVectors(target, playerThreatForward);
		float toPlayerThreat[3];
		GetClientAbsOrigin(target, toPlayerThreat);
		float myOrigin[3];
		GetClientAbsOrigin(actor, myOrigin);
		SubtractVectors(toPlayerThreat, myOrigin, toPlayerThreat);
		float threatRange = NormalizeVector(toPlayerThreat, toPlayerThreat);
		bool isBehindVictim = GetVectorDotProduct(playerThreatForward, toPlayerThreat) > Go_behindTolerance;
		bool isMovingTowardsVictim = true;
		if (IsLineOfFireClearEntity(actor, GetEyePosition(actor), target))
		{
			if (threatRange < Go_circleStrafeRange)
			{
				AimHeadTowards(myBot.GetBodyInterface(), WorldSpaceCenter(target), MANDATORY, 0.1, Address_Null, "Aim stab");
				if (!isBehindVictim)
				{
					float myForward[3];
					BasePlayer_EyeVectors(actor, myForward);
					float cross[3];
					GetVectorCrossProduct(playerThreatForward, myForward, cross);
					if (cross[2] < 0.0)
					{
						g_arrExtraButtons[actor].PressButtons(IN_MOVERIGHT, 0.1);
					}
					else
					{
						g_arrExtraButtons[actor].PressButtons(IN_MOVELEFT, 0.1);
					}
					if ((threatRange < 100.0) && !HasBackstabPotential(target))
					{
						isMovingTowardsVictim = false;
					}
				}
			}
			if (threatRange < GetStabRangeForTarget(target))
			{
				if (TF2_IsPlayerInCondition(actor, TFCond_Disguised))
				{
					if (redbots_manager_bot_backstab_skill.IntValue == 1)
					{
						if (GetEntProp(melee, Prop_Send, "m_bReadyToBackstab") != 0)
						{
							VS_PressFireButton(actor);
						}
					}
					else
					{
						if (isBehindVictim || HasBackstabPotential(target))
						{
							VS_PressFireButton(actor);
						}
					}
				}
				else
				{
					VS_PressFireButton(actor);
				}
			}
		}
		if (isMovingTowardsVictim)
		{
			m_pChasePath[actor].Update(myBot, target);
		}
	}
	else
	{
		int flag = FindBombNearestToHatch();
		if (flag != -1)
		{
			float bombPosition[3];
			bombPosition = GetAbsOrigin(flag);
			if (myBot.IsRangeGreaterThanEx(bombPosition, 200.0))
			{
				if (m_flRepathTime[actor] <= GetGameTime())
				{
					m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.9, 1.0);
					RepathToPos(actor, myBot, bombPosition);
				}
				m_pPath[actor].Update(myBot);
			}
		}
	}
	m_iAttackTarget[actor] = target;
	return action.Continue();
}

static Action CTFBotSpyLurkMvM_ShouldAttack(BehaviorAction action, INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	result = ANSWER_NO;
	return Plugin_Changed;
}

static Action CTFBotSpyLurkMvM_IsHindrance(BehaviorAction action, INextBot nextbot, int entity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	int me = action.Actor;
	if ((m_iAttackTarget[me] != -1) && nextbot.IsRangeLessThan(m_iAttackTarget[me], 300.0))
	{
		result = ANSWER_NO;
		return Plugin_Changed;
	}
	result = ANSWER_UNDEFINED;
	return Plugin_Changed;
}

stock float GetStabRangeForTarget(int target)
{
	return 75.0 * BaseAnimating_GetModelScale(target);
}

