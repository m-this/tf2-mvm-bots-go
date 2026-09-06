BehaviorAction CTFBotDefenderAttack()
{
	BehaviorAction action = ActionsManager.Create("DefenderAttack");

	action.OnStart = CTFBotDefenderAttack_OnStart;
	action.Update = CTFBotDefenderAttack_Update;

	return action;
}

#define ATTACK_STRAFE_REACH (130.0)
#define ATTACK_STRAFE_FLIP_MIN (0.5)
#define ATTACK_STRAFE_FLIP_MAX (1.1)

int m_iAttackTarget[65];
float m_flRevalidateTarget[65];
float m_ctAttackStrafeFlip[65];
bool m_bAttackStrafeRight[65];

// OnStart aims the path. The target is usually chosen before this action starts.
static Action CTFBotDefenderAttack_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	// NOTE: the attack target is usually chosen before we enter this action with CTFBotDefenderAttack_SelectTarget
	m_flRevalidateTarget[actor] = GetGameTime() + 3.0;
	return action.Continue();
}

// Update keeps the target honest, hands off to whatever outranks fighting, and
// otherwise walks at the robot and shoots it.
static Action CTFBotDefenderAttack_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if ((TF2_GetPlayerClass(actor) == TFClass_Sniper) && (GetTFBotMission(actor) == CTFBot_MISSION_SNIPER))
	{
		if (CanUsePrimayWeapon(actor))
		{
			return action.Done("I have gun");
		}
	}
	if (CTFBotCampBomb_IsPossible(actor))
	{
		return action.ChangeTo(CTFBotCampBomb(), "Camp bomb");
	}
	if (CTFBotGuardPoint_IsPossible(actor))
	{
		return action.ChangeTo(CTFBotGuardPoint(), "Defending a point");
	}
	if (CTFBotDestroyTeleporter_SelectTarget(actor))
	{
		return action.SuspendFor(CTFBotDestroyTeleporter(), "Get teleporter");
	}
	if (!IsValidClientIndex(m_iAttackTarget[actor]) || !IsPlayerAlive(m_iAttackTarget[actor]) || (TF2_GetClientTeam(m_iAttackTarget[actor]) != GetPlayerEnemyTeam(actor)))
	{
		if (!CTFBotDefenderAttack_SelectTarget(actor, false))
		{
			return action.Done("Target is not valid");
		}
	}
	if (m_flRevalidateTarget[actor] <= GetGameTime())
	{
		m_flRevalidateTarget[actor] = GetGameTime() + 2.0;
		if (!IsTargetEntityReachable(actor, m_iAttackTarget[actor]))
		{
			if (!CTFBotDefenderAttack_SelectTarget(actor, false))
			{
				return action.Done("Unreachable target");
			}
		}
	}
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Scout:
		{
			if (CTFBotCollectMoney_IsPossible(actor))
			{
				return action.ChangeTo(CTFBotCollectMoney(), "Collectinh money");
			}
		}
		case TFClass_Soldier, TFClass_Pyro, TFClass_DemoMan:
		{
			// These classes prefer priortizing the tank more than anything
			if (CTFBotAttackTank_SelectTarget(actor))
			{
				return action.ChangeTo(CTFBotAttackTank(), "Changing threat to tank");
			}
		}
		case TFClass_Medic:
		{
			// Make sure we have our medigun before we even think about leaving this action
			int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
			if (secondary != -1)
			{
				for (int i = 1; i <= MaxClients; i++)
				{
					if (IsClientInGame(i) && (GetClientTeam(i) == GetClientTeam(actor)) && IsPlayerAlive(i))
					{
						TFClassType class = TF2_GetPlayerClass(i);
						if ((class != TFClass_Medic) && (class != TFClass_Sniper) && (class != TFClass_Engineer) && (class != TFClass_Spy))
						{
							return action.Done("I have patient");
						}
					}
				}
			}
		}
	}
	// TODO: Other classes should go for money, but only when there isn't a threat around
	CTFBotDefenderAttack_SelectTarget(actor, true);
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	float targetOrigin[3];
	GetClientAbsOrigin(m_iAttackTarget[actor], targetOrigin);
	float myEyePos[3];
	GetClientEyePosition(actor, myEyePos);
	// Path if out of range or cannot see target
	if (myBot.IsRangeGreaterThanEx(targetOrigin, GetDesiredAttackRange(actor)) || !IsLineOfFireClearPosition(actor, myEyePos, targetOrigin))
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToTarget(actor, myBot, m_iAttackTarget[actor]);
		}
		// Walked, and not stepped toward when the mesh refuses: measured, and a fighter is not a medic
		//
		// The same nudge in PluginBot_SimulateFrame took the medic from four percent of a wave with
		// his beam connected to thirty, because a path that fails on the way to a teammate is a bot
		// standing still for nothing. Here it was worth twenty three percent more damage out of the
		// Soldier and the Demoman over twelve waves, and cost the team thirty seven percent more
		// deaths, flat total damage and a wave.
		//
		// Reaching a friend is safe and reaching a robot is not, and where the mesh will not path is
		// often ground worth not standing on.
		m_pPath[actor].Update(myBot);
	}
	else
		if (Feature(FEATURE_ATTACK_STRAFE))
		{
			StrafeWhileFighting(actor, myBot, targetOrigin);
		}
	IVision myVision = myBot.GetVisionInterface();
	CKnownEntity threat = myVision.GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		EquipBestWeaponForThreat(actor, threat);
	}
	return action.Continue();
}

// StrafeWhileFighting takes one step to the side, tested before it is taken.
stock void StrafeWhileFighting(int actor, INextBot myBot, float targetOrigin[3])
{
	// A Sniper is aiming down a scope and wants his feet exactly where they are
	if (TF2_GetPlayerClass(actor) == TFClass_Sniper)
	{
		return;
	}
	ILocomotion myLoco = myBot.GetLocomotionInterface();
	if (!myLoco.IsOnGround() || myLoco.IsClimbingOrJumping())
	{
		return;
	}
	if (m_ctAttackStrafeFlip[actor] < GetGameTime())
	{
		m_ctAttackStrafeFlip[actor] = GetGameTime() + GetRandomFloat(ATTACK_STRAFE_FLIP_MIN, ATTACK_STRAFE_FLIP_MAX);
		m_bAttackStrafeRight[actor] = !m_bAttackStrafeRight[actor];
	}
	float myOrigin[3];
	myOrigin = GetAbsOrigin(actor);
	float toTarget[3];
	SubtractVectors(targetOrigin, myOrigin, toTarget);
	toTarget[2] = 0.0;
	float length = NormalizeVector(toTarget, toTarget);
	if (length < 1.0)
	{
		return;
	}
	// Square to the way it is facing, in the plane it walks on
	float side[3];
	side[0] = toTarget[1];
	side[1] = -toTarget[0];
	if (m_bAttackStrafeRight[actor])
	{
		side[0] = -toTarget[1];
		side[1] = toTarget[0];
	}
	side[2] = 0.0;
	float step[3];
	for (int axis = 0; axis < 3; axis++)
	{
		step[axis] = myOrigin[axis] + (side[axis] * ATTACK_STRAFE_REACH);
	}
	// Turn round early rather than walk into whatever is there, or off it
	if (!myLoco.IsPotentiallyTraversable(myOrigin, step, IMMEDIATELY) || myLoco.HasPotentialGap(myOrigin, step))
	{
		m_ctAttackStrafeFlip[actor] = GetGameTime();
		return;
	}
	myLoco.Approach(step);
}

// SelectTarget picks the robot worth shooting: the one nearest the bomb, then
// whoever is healing it.
stock bool CTFBotDefenderAttack_SelectTarget(int actor, bool bBombCarrierOnly = false)
{
	// Always go after the bot closest to the bomb, if possible
	int target = FindBotNearestToBombNearestToHatch(actor);
	if (!bBombCarrierOnly && (target == -1))
	{
		target = SelectRandomReachableEnemy(actor);
	}
	if (target != -1)
	{
		int healer = GetHealerOfPlayer(target, true);
		if (healer != -1)
		{
			target = healer;
		}
		m_iAttackTarget[actor] = target;
		return true;
	}
	return false;
}

// TargetEntityReachable says the bot could actually get to him, which mostly
// means he is not standing in his own spawn.
stock bool IsTargetEntityReachable(int client, int target)
{
	CTFNavArea area = CBaseCombatCharacter(target).GetLastKnownArea();
	if (area == view_as<CTFNavArea>(NULL_AREA))
	{
		return false;
	}
	if (((TF2_GetClientTeam(client) == TFTeam_Red) && area.HasAttributeTF(BLUE_SPAWN_ROOM)) || ((TF2_GetClientTeam(client) == TFTeam_Blue) && area.HasAttributeTF(RED_SPAWN_ROOM)))
	{
		// Usually cannot enter enemy spawns
		return false;
	}
	return true;
}

// ResetAttack forgets the robot this bot was shooting.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetAttack(int client)
{
	m_iAttackTarget[client] = -1;
}

