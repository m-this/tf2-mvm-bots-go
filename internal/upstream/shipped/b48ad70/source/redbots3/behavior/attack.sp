int m_iAttackTarget[MAXPLAYERS + 1];
float m_flRevalidateTarget[MAXPLAYERS + 1];

BehaviorAction CTFBotDefenderAttack()
{
	BehaviorAction action = ActionsManager.Create("DefenderAttack");
	
	action.OnStart = CTFBotDefenderAttack_OnStart;
	action.Update = CTFBotDefenderAttack_Update;
	
	return action;
}

static Action CTFBotDefenderAttack_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	
	//NOTE: the attack target is usually chosen before we enter this action with CTFBotDefenderAttack_SelectTarget
	
	m_flRevalidateTarget[actor] = GetGameTime() + 3.0;
	
	return action.Continue();
}

static Action CTFBotDefenderAttack_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (TF2_GetPlayerClass(actor) == TFClass_Sniper && GetTFBotMission(actor) == CTFBot_MISSION_SNIPER)
	{
		if (CanUsePrimayWeapon(actor))
		{
			//We can snipe again
			return action.Done("I have gun");
		}
	}
	
	if (CTFBotCampBomb_IsPossible(actor))
		return action.ChangeTo(CTFBotCampBomb(), "Camp bomb");
	
	if (CTFBotGuardPoint_IsPossible(actor))
		return action.ChangeTo(CTFBotGuardPoint(), "Defending a point");
	
	if (CTFBotDestroyTeleporter_SelectTarget(actor))
		return action.SuspendFor(CTFBotDestroyTeleporter(), "Get teleporter");
	
	if (!IsValidClientIndex(m_iAttackTarget[actor])
	|| !IsPlayerAlive(m_iAttackTarget[actor])
	|| TF2_GetClientTeam(m_iAttackTarget[actor]) != GetPlayerEnemyTeam(actor))
	{
		if (!CTFBotDefenderAttack_SelectTarget(actor))
			return action.Done("Target is not valid");
	}
	
	if (m_flRevalidateTarget[actor] <= GetGameTime())
	{
		m_flRevalidateTarget[actor] = GetGameTime() + 2.0;
	
		//Need new target.
		if (!IsTargetEntityReachable(actor, m_iAttackTarget[actor]))
			if (!CTFBotDefenderAttack_SelectTarget(actor))
				return action.Done("Unreachable target");
	}
	
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Scout:
		{
			//Scouts primarily prefer to get money
			if (CTFBotCollectMoney_IsPossible(actor))
				return action.ChangeTo(CTFBotCollectMoney(), "Collectinh money");
		}
		case TFClass_Soldier, TFClass_Pyro, TFClass_DemoMan:
		{
			//These classes prefer priortizing the tank more than anything
			if (CTFBotAttackTank_SelectTarget(actor))
				return action.ChangeTo(CTFBotAttackTank(), "Changing threat to tank");
		}
		case TFClass_Medic:
		{
			//Make sure we have our medigun before we even think about leaving this action
			int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
			
			if (secondary != -1)
			{
				for (int i = 1; i <= MaxClients; i++)
				{
					if (IsClientInGame(i) && GetClientTeam(i) == GetClientTeam(actor) && IsPlayerAlive(i))
					{
						TFClassType class = TF2_GetPlayerClass(i);
						
						if (class != TFClass_Medic && class != TFClass_Sniper && class != TFClass_Engineer && class != TFClass_Spy)
						{
							//We have someone we'd prefer to heal
							return action.Done("I have patient");
						}
					}
				}
			}
		}
	}
	
	//TODO: Other classes should go for money, but only when there isn't a threat around
	
	CTFBotDefenderAttack_SelectTarget(actor, true);
	
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	float targetOrigin[3]; GetClientAbsOrigin(m_iAttackTarget[actor], targetOrigin);
	float myEyePos[3]; GetClientEyePosition(actor, myEyePos);
	
	//Path if out of range or cannot see target
	if (myBot.IsRangeGreaterThanEx(targetOrigin, GetDesiredAttackRange(actor)) || !IsLineOfFireClearPosition(actor, myEyePos, targetOrigin))
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToTarget(actor, myBot, m_iAttackTarget[actor]);
		}
		
		/* Walked, and not stepped toward when the mesh refuses: measured, and a fighter is not a medic
		
		The same nudge in PluginBot_SimulateFrame took the medic from four percent of a wave with
		his beam connected to thirty, because a path that fails on the way to a teammate is a bot
		standing still for nothing. Here it was worth twenty three percent more damage out of the
		Soldier and the Demoman over twelve waves, and cost the team thirty seven percent more
		deaths, flat total damage and a wave.
		
		Reaching a friend is safe and reaching a robot is not, and where the mesh will not path is
		often ground worth not standing on. */
		m_pPath[actor].Update(myBot);
	}
	else if (Feature(FEATURE_ATTACK_STRAFE))
	{
		StrafeWhileFighting(actor, myBot, targetOrigin);
	}
	
	IVision myVision = myBot.GetVisionInterface();
	CKnownEntity threat = myVision.GetPrimaryKnownThreat(false);
	
	if (threat)
	{
		//We have a threat, prepare to fight it
		EquipBestWeaponForThreat(actor, threat);
	}
	
	return action.Continue();
}

/* Sidestepping while it shoots, because a bot that has arrived stops moving entirely

The path above runs while the target is too far off or behind cover, and nothing runs once it is
neither. So the bot walks up, plants its feet, and stands still for the rest of the fight, which is
the one thing every guide about this game tells a person not to do. A stationary target is what a
robot's aim was written for, and the projectile classes among them do not miss one.

Sidestepping costs nothing in accuracy here. Projectiles do not inherit the shooter's velocity, and
the head is aimed by different code from the code that moves the feet, so a bot that strafes shoots
exactly as well as one that stands.

Approach and not a path: this is a step to one side, not a journey, and computing a path for every
flip of it would be a nav mesh search several times a second per bot, for a distance the bot covers
in a quarter of one.

The step is tested before it is taken. Locomotion stops itself walking into a wall; it will happily
walk off a ledge, and a Demoman who sidesteps into the pit on Rottenburg has solved the wrong
problem. */
#define ATTACK_STRAFE_REACH		130.0
#define ATTACK_STRAFE_FLIP_MIN	0.5
#define ATTACK_STRAFE_FLIP_MAX	1.1

static float m_ctAttackStrafeFlip[MAXPLAYERS + 1];
static bool m_bAttackStrafeRight[MAXPLAYERS + 1];

static void StrafeWhileFighting(int actor, INextBot myBot, const float targetOrigin[3])
{
	//A Sniper is aiming down a scope and wants his feet exactly where they are
	if (TF2_GetPlayerClass(actor) == TFClass_Sniper)
		return;
	
	ILocomotion myLoco = myBot.GetLocomotionInterface();
	
	if (!myLoco.IsOnGround() || myLoco.IsClimbingOrJumping())
		return;
	
	if (m_ctAttackStrafeFlip[actor] < GetGameTime())
	{
		m_ctAttackStrafeFlip[actor] = GetGameTime() + GetRandomFloat(ATTACK_STRAFE_FLIP_MIN, ATTACK_STRAFE_FLIP_MAX);
		m_bAttackStrafeRight[actor] = !m_bAttackStrafeRight[actor];
	}
	
	float myOrigin[3]; myOrigin = GetAbsOrigin(actor);
	float toTarget[3]; SubtractVectors(targetOrigin, myOrigin, toTarget);
	
	toTarget[2] = 0.0;
	
	if (NormalizeVector(toTarget, toTarget) < 1.0)
		return;
	
	//Square to the way it is facing, in the plane it walks on
	float side[3];
	side[0] = m_bAttackStrafeRight[actor] ? -toTarget[1] : toTarget[1];
	side[1] = m_bAttackStrafeRight[actor] ? toTarget[0] : -toTarget[0];
	side[2] = 0.0;
	
	float step[3];
	
	for (int axis = 0; axis < 3; axis++)
		step[axis] = myOrigin[axis] + side[axis] * ATTACK_STRAFE_REACH;
	
	//Turn round early rather than walk into whatever is there, or off it
	if (!myLoco.IsPotentiallyTraversable(myOrigin, step, IMMEDIATELY) || myLoco.HasPotentialGap(myOrigin, step))
	{
		m_ctAttackStrafeFlip[actor] = GetGameTime();
		
		return;
	}
	
	myLoco.Approach(step);
}

bool CTFBotDefenderAttack_SelectTarget(int actor, bool bBombCarrierOnly = false)
{
	//Always go after the bot closest to the bomb, if possible
	int target = FindBotNearestToBombNearestToHatch(actor);
	
	//No bomb in play, just find random target
	if (!bBombCarrierOnly && target == -1)
		target = SelectRandomReachableEnemy(actor);
	
	//Found a valid target, update
	if (target != -1)
	{
		//Go after the healer first
		int healer = GetHealerOfPlayer(target, true);
		
		if (healer != -1)
			target = healer;
		
		m_iAttackTarget[actor] = target;
		return true;
	}
	
	return false;
}

static bool IsTargetEntityReachable(int client, int target)
{
	CTFNavArea area = view_as<CTFNavArea>(CBaseCombatCharacter(target).GetLastKnownArea());
	
	if (area == NULL_AREA)
		return false;
	
	if ((TF2_GetClientTeam(client) == TFTeam_Red && area.HasAttributeTF(BLUE_SPAWN_ROOM))
	|| (TF2_GetClientTeam(client) == TFTeam_Blue && area.HasAttributeTF(RED_SPAWN_ROOM)))
	{
		//Usually cannot enter enemy spawns
		return false;
	}
	
	return true;
}