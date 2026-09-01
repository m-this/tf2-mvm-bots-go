BehaviorAction CTFBotMvMEngineerIdle()
{
	BehaviorAction action = ActionsManager.Create("DefenderEngineerIdle");

	action.OnStart = CTFBotMvMEngineerIdle_OnStart;
	action.Update = CTFBotMvMEngineerIdle_Update;
	action.OnEnd = CTFBotMvMEngineerIdle_OnEnd;
	action.OnMoveToSuccess = CTFBotMvMEngineerIdle_OnMoveToSuccess;

	return action;
}

#define Go_Slots (65)

#define SENTRY_WATCH_BOMB_RANGE (400.0)

#define NEST_ADVANCE_MARGIN (600.0)
#define NEST_ADVANCE_COOLDOWN (45.0)
#define NEST_ADVANCE_RECHECK (10.0)

#define NEST_RELOCATE_HAUL_TIME (20.0)

#define CARRY_GIVE_UP_TIME (25.0)

#define SENTRY_UNDER_FIRE_TIME (3.0)

#define ENGINEER_STALL_REPORT (10.0)

#define RANGE_REPAIR_PATIENCE (3.0)

#define NEST_RELOCATE_EVAL_INTERVAL (0.1)

float m_ctSentrySafe[65];
float m_ctAdvanceAgain[65];
float m_ctSentryCooldown[65];
float m_ctDispenserSafe[65];
float m_ctDispenserCooldown[65];
float m_ctFindNestHint[65];
float m_ctAdvanceNestSpot[65];
float m_ctRecomputePathMvMEngiIdle[65];
bool g_bGoingToGrabBuilding[65];
int m_hBuildingToGrab[65];
CNavArea m_aNestAreaBeforeHaul[65];
CNavArea m_aNestAreaBeforeRelocate[65];
float m_ctNestRelocateDeadline[65];
float m_ctCarryDeadline[65];
float m_ctSentryUnderFire[65];
int m_iSentryHealthLast[65];
float m_ctEngineerStallReport[65];
int m_iRangeRepairHealth[65];
float m_ctRangeRepairSince[65];
int m_iRangeRepairStalls[65];
int m_iNestRelocateEvalNext;
Handle m_hNestRelocateEvalTimer;

stock bool IsWorthAdvancingTo(CNavArea held, CNavArea candidate)
{
	if ((candidate == NULL_AREA) || (candidate == held))
	{
		return false;
	}
	if (held == NULL_AREA)
	{
		return true;
	}
	return GetTravelDistanceToBombTarget(candidate) < (GetTravelDistanceToBombTarget(held) - NEST_ADVANCE_MARGIN);
}

static Action CTFBotMvMEngineerIdle_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_aNestAreaBeforeHaul[actor] = NULL_AREA;
	m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
	m_ctNestRelocateDeadline[actor] = -1.0;
	m_iSentryHealthLast[actor] = 0;
	CTFBotMvMEngineerIdle_ResetProperties(actor);
	return action.Continue();
}

stock void ReportEngineerStall(int actor, const char[] where)
{
	if (m_ctEngineerStallReport[actor] > GetGameTime())
	{
		return;
	}
	m_ctEngineerStallReport[actor] = GetGameTime() + ENGINEER_STALL_REPORT;
	PrintToServer("[defenderbots] engineer %N has no sentry at %.1f: %s (nest %s, carrying %s, grabbing %s)", actor, GetGameTime(), where, (Go_nestAreaOf(actor) == NULL_AREA ? "none" : "held"), (TF2_IsCarryingObject(actor) ? "yes" : "no"), (g_bGoingToGrabBuilding[actor] ? "yes" : "no"));
}

stock CNavArea Go_nestAreaOf(int actor)
{
	return m_aNestArea[actor];
}

static Action CTFBotMvMEngineerIdle_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	int sentry = HasObjectOfType(actor, TFObject_Sentry, TFObjectMode_None);
	int dispenser = HasObjectOfType(actor, TFObject_Dispenser, TFObjectMode_None);
	int sentryStanding = GetObjectOfType(actor, TFObject_Sentry);
	bool stalled = (sentryStanding == INVALID_ENT_REFERENCE) && (GameRules_GetRoundState() == RoundState_RoundRunning);
	bool bShouldAdvance = CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(actor);
	int buster = -1;
	if (!g_bGoingToGrabBuilding[actor])
	{
		int found;
		bool haul = ShouldHaulFromSentryBuster(actor, sentry, found);
		buster = found;
		if (haul)
		{
			CNavArea retreat = PickBusterRetreatArea(sentry, buster);
			if (retreat != NULL_AREA)
			{
				if (redbots_manager_debug_actions.BoolValue)
				{
					PrintToServer("CTFBotMvMEngineerIdle_Update: HAUL FROM BUSTER");
				}
				BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_INCOMING);
				m_aNestAreaBeforeHaul[actor] = m_aNestArea[actor];
				CTFBotMvMEngineerIdle_ResetProperties(actor);
				m_aNestArea[actor] = retreat;
				g_bGoingToGrabBuilding[actor] = true;
				m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);
				g_arrPluginBot[actor].SetPathGoalEntity(sentry);
				return action.Continue();
			}
		}
	}
	if ((m_aNestAreaBeforeHaul[actor] != NULL_AREA) && (buster == -1) && !g_bGoingToGrabBuilding[actor] && (sentry != INVALID_ENT_REFERENCE))
	{
		CNavArea home = m_aNestAreaBeforeHaul[actor];
		m_aNestAreaBeforeHaul[actor] = NULL_AREA;
		CTFBotMvMEngineerIdle_ResetProperties(actor);
		m_aNestArea[actor] = home;
		g_bGoingToGrabBuilding[actor] = true;
		m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);
		g_arrPluginBot[actor].SetPathGoalEntity(sentry);
		return action.Continue();
	}
	if ((m_aNestAreaRelocate[actor] != NULL_AREA) && (m_aNestAreaBeforeHaul[actor] == NULL_AREA) && !g_bGoingToGrabBuilding[actor] && !TF2_IsCarryingObject(actor) && ((sentry == INVALID_ENT_REFERENCE) || !TF2_IsBuilding(sentry)))
	{
		CNavArea destination = m_aNestAreaRelocate[actor];
		m_aNestAreaRelocate[actor] = NULL_AREA;
		if (sentry == INVALID_ENT_REFERENCE)
		{
			m_aNestArea[actor] = destination;
		}
		else
		{
			if (redbots_manager_debug_actions.BoolValue)
			{
				PrintToServer("CTFBotMvMEngineerIdle_Update: RELOCATE NEST");
			}
			m_aNestAreaBeforeRelocate[actor] = m_aNestArea[actor];
			CTFBotMvMEngineerIdle_ResetProperties(actor);
			m_ctNestRelocateDeadline[actor] = GetGameTime() + NEST_RELOCATE_HAUL_TIME;
			m_aNestArea[actor] = destination;
			g_bGoingToGrabBuilding[actor] = true;
			m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);
			g_arrPluginBot[actor].SetPathGoalEntity(sentry);
			DetonateObjectOfType(actor, TFObject_Dispenser);
			return action.Continue();
		}
	}
	if (bShouldAdvance && !g_bGoingToGrabBuilding[actor])
	{
		CNavArea candidate = PickBuildArea(actor);
		if ((m_ctAdvanceAgain[actor] > GetGameTime()) || !IsWorthAdvancingTo(m_aNestArea[actor], candidate))
		{
			m_ctAdvanceNestSpot[actor] = GetGameTime() + NEST_ADVANCE_RECHECK;
		}
		else
		{
			if (redbots_manager_debug_actions.BoolValue)
			{
				PrintToServer("CTFBotMvMEngineerIdle_Update: ADVANCE");
			}
			CTFBotMvMEngineerIdle_ResetProperties(actor);
			m_aNestArea[actor] = candidate;
			m_ctAdvanceAgain[actor] = GetGameTime() + NEST_ADVANCE_COOLDOWN;
			if ((sentry != INVALID_ENT_REFERENCE) && (m_aNestArea[actor] != NULL_AREA))
			{
				g_bGoingToGrabBuilding[actor] = true;
				m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);
				g_arrPluginBot[actor].SetPathGoalEntity(sentry);
			}
		}
	}
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	ILocomotion myLoco = myNextbot.GetLocomotionInterface();
	if ((m_aNestAreaBeforeRelocate[actor] != NULL_AREA) && (m_ctNestRelocateDeadline[actor] > 0.0) && (GetGameTime() > m_ctNestRelocateDeadline[actor]))
	{
		m_ctNestRelocateDeadline[actor] = -1.0;
		if (TF2_IsCarryingObject(actor))
		{
			CNavArea here = TheNavMesh.GetNearestNavArea(GetAbsOrigin(actor), false, 500.0, false, true, TEAM_ANY);
			if (here != NULL_AREA)
			{
				m_aNestArea[actor] = here;
			}
		}
		else
		{
			m_aNestArea[actor] = m_aNestAreaBeforeRelocate[actor];
			g_bGoingToGrabBuilding[actor] = false;
			m_hBuildingToGrab[actor] = INVALID_ENT_REFERENCE;
		}
		m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
	}
	if (g_bGoingToGrabBuilding[actor])
	{
		int building = EntRefToEntIndex(m_hBuildingToGrab[actor]);
		if (!TF2_IsCarryingObject(actor))
		{
			m_ctCarryDeadline[actor] = 0.0;
		}
		else
			if (m_ctCarryDeadline[actor] <= 0.0)
			{
				m_ctCarryDeadline[actor] = GetGameTime() + CARRY_GIVE_UP_TIME;
			}
			else
				if (GetGameTime() > m_ctCarryDeadline[actor])
				{
					CNavArea here = TheNavMesh.GetNearestNavArea(GetAbsOrigin(actor), false, 500.0, false, true, TEAM_ANY);
					m_ctCarryDeadline[actor] = 0.0;
					LogBuildFailure(actor, "carry", "held it too long, putting it down here");
					if (here != NULL_AREA)
					{
						m_aNestArea[actor] = here;
					}
				}
		if (building == INVALID_ENT_REFERENCE)
		{
			g_bGoingToGrabBuilding[actor] = false;
			m_hBuildingToGrab[actor] = INVALID_ENT_REFERENCE;
			m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
			m_ctNestRelocateDeadline[actor] = -1.0;
			if (redbots_manager_debug_actions.BoolValue)
			{
				PrintToServer("CTFBotMvMEngineerIdle_Update: g_bGoingToGrabBuilding : building %i | m_aNestArea %x", building, m_aNestArea[actor]);
			}
			DetonateObjectOfType(actor, TFObject_Sentry);
			DetonateObjectOfType(actor, TFObject_Dispenser);
			g_arrPluginBot[actor].bPathing = false;
			if (stalled)
			{
				ReportEngineerStall(actor, "the building he was fetching is gone");
			}
			return action.Continue();
		}
		UpdateLookAroundForEnemies(actor, false);
		if (!TF2_IsCarryingObject(actor))
		{
			float flDistanceToBuilding = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(building));
			if (flDistanceToBuilding < 90.0)
			{
				EquipWeaponSlot(actor, TFWeaponSlot_Melee);
				AimHeadTowards(myBody, WorldSpaceCenter(building), CRITICAL, 1.0, Address_Null, "Grab building");
				VS_PressAltFireButton(actor);
			}
		}
		else
		{
			if (m_aNestArea[actor] != NULL_AREA)
			{
				float center[3];
				m_aNestArea[actor].GetCenter(center);
				g_arrPluginBot[actor].SetPathGoalVector(center);
				float flDistanceToGoal = GetVectorDistance(GetAbsOrigin(actor), center);
				if (flDistanceToGoal < 200.0)
				{
					if (!myLoco.IsStuck())
					{
						g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
					}
					if (flDistanceToGoal < 70.0)
					{
						int objBeingBuilt = TF2_GetCarriedObject(actor);
						if (objBeingBuilt == -1)
						{
							return action.Continue();
						}
						bool bPlacementOK = IsPlacementOK(objBeingBuilt);
						VS_PressFireButton(actor);
						if (!bPlacementOK && myBody.IsHeadAimingOnTarget() && (myBody.GetHeadSteadyDuration() > 0.6))
						{
							m_aNestArea[actor] = PickBuildArea(actor);
						}
						else
						{
							g_bGoingToGrabBuilding[actor] = false;
							m_hBuildingToGrab[actor] = INVALID_ENT_REFERENCE;
							m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
							m_ctNestRelocateDeadline[actor] = -1.0;
							g_arrPluginBot[actor].bPathing = false;
						}
					}
				}
			}
		}
		g_arrPluginBot[actor].bPathing = true;
		return action.Continue();
	}
	if (((m_aNestArea[actor] == NULL_AREA) || bShouldAdvance) || (sentry == INVALID_ENT_REFERENCE))
	{
		if ((m_ctFindNestHint[actor] > 0.0) && (m_ctFindNestHint[actor] > GetGameTime()))
		{
			return action.Continue();
		}
		m_ctFindNestHint[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
		m_aNestArea[actor] = PickBuildArea(actor);
	}
	if (bShouldAdvance)
	{
		if (stalled)
		{
			ReportEngineerStall(actor, "advancing the nest");
		}
		return action.Continue();
	}
	UpdateSentryUnderFire(actor, sentry);
	if (sentry != -1)
	{
		if (((m_ctSentrySafe[actor] > GetGameTime()) || (m_ctSentryUnderFire[actor] > GetGameTime())) && !g_bGoingToGrabBuilding[actor])
		{
			int mySecondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
			if ((mySecondary != -1) && (TF2Util_GetWeaponID(mySecondary) == TF_WEAPON_LASER_POINTER) && myNextbot.IsRangeLessThan(sentry, 180.0))
			{
				CKnownEntity threat = myNextbot.GetVisionInterface().GetPrimaryKnownThreat(false);
				if (threat != 0)
				{
					int iThreat = threat.GetEntity();
					bool defending = m_ctSentryUnderFire[actor] > GetGameTime();
					if ((defending || (GetVectorDistance(GetAbsOrigin(sentry), GetAbsOrigin(iThreat)) > SENTRY_MAX_RANGE)) && IsLineOfFireClearEntity(actor, GetEyePosition(actor), iThreat))
					{
						AimHeadTowards(myBody, WorldSpaceCenter(iThreat), MANDATORY, 0.1, Address_Null, "Aiming!");
						TF2Util_SetPlayerActiveWeapon(actor, mySecondary);
						if (myBody.IsHeadAimingOnTarget() && (GetEntProp(sentry, Prop_Send, "m_bPlayerControlled") != 0))
						{
							OSLib_RunScriptCode(actor, _, _, "self.PressFireButton(0.1);self.PressAltFireButton(0.1)");
						}
						g_arrPluginBot[actor].bPathing = false;
						return action.Continue();
					}
				}
			}
		}
	}
	if ((m_aNestArea[actor] == NULL_AREA) && stalled)
	{
		ReportEngineerStall(actor, "no nest area to build on");
	}
	if (m_aNestArea[actor] != NULL_AREA)
	{
		if (sentry != INVALID_ENT_REFERENCE)
		{
			if (IsSentrySafe(sentry))
			{
				m_ctSentrySafe[actor] = GetGameTime() + 3.0;
			}
			m_ctSentryCooldown[actor] = GetGameTime() + 3.0;
		}
		else
		{
			if (m_ctSentryCooldown[actor] >= GetGameTime())
			{
				ReportEngineerStall(actor, "waiting out the rebuild cooldown");
			}
			if (m_ctSentryCooldown[actor] < GetGameTime())
			{
				m_ctSentryCooldown[actor] = GetGameTime() + 3.0;
				return action.SuspendFor(CTFBotMvMEngineerBuildSentrygun(), "No sentry - building a new one");
			}
		}
		if (sentry != INVALID_ENT_REFERENCE)
		{
			if (dispenser != INVALID_ENT_REFERENCE)
			{
				if (m_ctSentrySafe[actor] < GetGameTime())
				{
					m_ctDispenserCooldown[actor] = GetGameTime() + 3.0;
				}
			}
			else
			{
				if ((m_ctDispenserCooldown[actor] < GetGameTime()) && (m_ctSentrySafe[actor] > GetGameTime()))
				{
					m_ctDispenserCooldown[actor] = GetGameTime() + 3.0;
					return action.SuspendFor(CTFBotMvMEngineerBuildDispenser(), "Sentry safe, No dispenser - building one");
				}
			}
		}
	}
	if ((m_ctSentrySafe[actor] > GetGameTime()) && !g_bGoingToGrabBuilding[actor] && ShouldBuildTeleporter(actor))
	{
		return action.SuspendFor(CTFBotMvMEngineerBuildTeleporter(), "Nest is up, building a teleporter");
	}
	if ((m_ctSentrySafe[actor] > GetGameTime()) && !g_bGoingToGrabBuilding[actor] && ShouldBuildDisposable(actor))
	{
		return action.SuspendFor(CTFBotMvMEngineerBuildDisposable(), "Nest is up, standing a mini beside it");
	}
	bool sentryWantsMetal = (sentry != INVALID_ENT_REFERENCE) && SentryNeedsMetal(sentry);
	if ((dispenser != INVALID_ENT_REFERENCE) && !sentryWantsMetal && (m_ctSentrySafe[actor] > GetGameTime()))
	{
		if ((TF2_GetUpgradeLevel(dispenser) < 3) || (BaseEntity_GetHealth(dispenser) < TF2Util_GetEntityMaxHealth(dispenser)))
		{
			float dist = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(dispenser));
			if (m_ctRecomputePathMvMEngiIdle[actor] < GetGameTime())
			{
				m_ctRecomputePathMvMEngiIdle[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
				float dir[3];
				SubtractVectors(GetAbsAngles(dispenser), GetAbsOrigin(actor), dir);
				NormalizeVector(dir, dir);
				float goal[3];
				goal = GetAbsOrigin(dispenser);
				goal[0] -= 50.0 * dir[0];
				goal[1] -= 50.0 * dir[1];
				goal[2] -= 50.0 * dir[2];
				if (IsPathToVectorPossible(actor, goal))
				{
					g_arrPluginBot[actor].SetPathGoalVector(goal);
				}
				else
				{
					g_arrPluginBot[actor].SetPathGoalEntity(sentry);
				}
				g_arrPluginBot[actor].bPathing = true;
			}
			if (dist < 90.0)
			{
				if (!myLoco.IsStuck())
				{
					g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
				}
				EquipWeaponSlot(actor, TFWeaponSlot_Melee);
				UpdateLookAroundForEnemies(actor, false);
				AimHeadTowards(myBody, WorldSpaceCenter(dispenser), CRITICAL, 1.0, Address_Null, "Work on my Dispenser");
				VS_PressFireButton(actor);
			}
			return action.Continue();
		}
	}
	if (sentry != INVALID_ENT_REFERENCE)
	{
		float dist = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(sentry));
		if (!SentryNeedsMetal(sentry))
		{
			EquipWeaponSlot(actor, TFWeaponSlot_Primary);
			UpdateLookAroundForEnemies(actor, true);
		}
		else
			if (CanRepairFromRange(actor, sentry, dist))
			{
				EquipWeaponSlot(actor, TFWeaponSlot_Primary);
				AimHeadTowards(myBody, WorldSpaceCenter(sentry), CRITICAL, 1.0, Address_Null, "Repair my Sentry from here");
				if (myBody.IsHeadAimingOnTarget())
				{
					VS_PressFireButton(actor);
				}
				g_arrPluginBot[actor].bPathing = false;
				return action.Continue();
			}
		if (m_ctRecomputePathMvMEngiIdle[actor] < GetGameTime())
		{
			m_ctRecomputePathMvMEngiIdle[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
			float vTurretAngles[3];
			GetTurretAngles(sentry, vTurretAngles);
			float dir[3];
			GetAngleVectors(vTurretAngles, dir, NULL_VECTOR, NULL_VECTOR);
			float goal[3];
			goal = GetAbsOrigin(sentry);
			goal[0] -= 50.0 * dir[0];
			goal[1] -= 50.0 * dir[1];
			goal[2] -= 50.0 * dir[2];
			if (IsPathToVectorPossible(actor, goal))
			{
				g_arrPluginBot[actor].SetPathGoalVector(goal);
			}
			else
			{
				g_arrPluginBot[actor].SetPathGoalEntity(sentry);
			}
			g_arrPluginBot[actor].bPathing = true;
		}
		if ((dist < 90.0) && SentryNeedsMetal(sentry))
		{
			if (!myLoco.IsStuck())
			{
				g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
			}
			EquipWeaponSlot(actor, TFWeaponSlot_Melee);
			UpdateLookAroundForEnemies(actor, false);
			AimHeadTowards(myBody, WorldSpaceCenter(sentry), CRITICAL, 1.0, Address_Null, "Work on my Sentry");
			VS_PressFireButton(actor);
		}
	}
	return action.Continue();
}

stock void UpdateSentryUnderFire(int actor, int sentry)
{
	if (sentry == INVALID_ENT_REFERENCE)
	{
		m_iSentryHealthLast[actor] = 0;
		return;
	}
	int health = BaseEntity_GetHealth(sentry);
	if ((m_iSentryHealthLast[actor] > 0) && (health < m_iSentryHealthLast[actor]))
	{
		m_ctSentryUnderFire[actor] = GetGameTime() + SENTRY_UNDER_FIRE_TIME;
	}
	m_iSentryHealthLast[actor] = health;
}

stock bool ShouldHaulFromSentryBuster(int actor, int sentry, int &buster)
{
	buster = 0;
	buster = -1;
	if (sentry == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (GameRules_GetRoundState() != RoundState_RoundRunning)
	{
		return false;
	}
	if (TF2_IsCarryingObject(actor) || TF2_IsBuilding(sentry))
	{
		return false;
	}
	buster = FindSentryBusterNear(GetAbsOrigin(sentry), GetPlayerEnemyTeam(actor), BUSTER_HAUL_RANGE);
	if (buster == -1)
	{
		return false;
	}
	if (TF2_IsMiniBuilding(sentry))
	{
		return false;
	}
	if (GetVectorDistance(GetAbsOrigin(sentry), WorldSpaceCenter(buster)) < BUSTER_FLEE_RANGE)
	{
		return false;
	}
	return true;
}

stock bool SentryNeedsMetal(int sentry)
{
	if (TF2_IsBuilding(sentry))
	{
		return true;
	}
	if (BaseEntity_GetHealth(sentry) < TF2Util_GetEntityMaxHealth(sentry))
	{
		return true;
	}
	if (!TF2_IsMiniBuilding(sentry) && (TF2_GetUpgradeLevel(sentry) < 3))
	{
		return true;
	}
	return GetEntProp(sentry, Prop_Send, "m_iAmmoShells") < 100;
}

stock int RangeRepairStallsOf(int client)
{
	return m_iRangeRepairStalls[client];
}

stock void NoteRangeRepair(int actor, int sentry)
{
	int health = BaseEntity_GetHealth(sentry);
	float now = GetGameTime();
	if ((health > m_iRangeRepairHealth[actor]) || (m_ctRangeRepairSince[actor] <= 0.0))
	{
		m_iRangeRepairHealth[actor] = health;
		m_ctRangeRepairSince[actor] = now;
		return;
	}
	m_iRangeRepairHealth[actor] = health;
	if ((now - m_ctRangeRepairSince[actor]) < RANGE_REPAIR_PATIENCE)
	{
		return;
	}
	m_ctRangeRepairSince[actor] = now;
	m_iRangeRepairStalls[actor]++;
	LogBuildFailure(actor, "repairing at range", "three seconds of bolts and the sentry gained nothing");
}

stock void ForgetRangeRepair(int actor)
{
	m_iRangeRepairHealth[actor] = 0;
	m_ctRangeRepairSince[actor] = 0.0;
}

stock bool CanRepairFromRange(int actor, int sentry, float dist)
{
	if (!TF2_IsRescueRangerEquipped(actor))
	{
		return false;
	}
	if (BaseEntity_GetHealth(sentry) >= TF2Util_GetEntityMaxHealth(sentry))
	{
		return false;
	}
	if (dist < 200.0)
	{
		return false;
	}
	if (dist > SENTRY_MAX_RANGE)
	{
		return false;
	}
	if (GetEntProp(actor, Prop_Data, "m_iAmmo", 3, 4) < 30)
	{
		return false;
	}
	if (!IsLineOfFireClearEntity(actor, GetEyePosition(actor), sentry))
	{
		ForgetRangeRepair(actor);
		return false;
	}
	NoteRangeRepair(actor, sentry);
	return true;
}

static void CTFBotMvMEngineerIdle_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
}

static Action CTFBotMvMEngineerIdle_OnMoveToSuccess(BehaviorAction action, int actor, any path, ActionDesiredResult result)
{
	CBaseNPC_GetNextBotOfEntity(actor).GetLocomotionInterface().ClearStuckStatus("Arrived at goal");
	return action.TryContinue();
}

stock void CTFBotMvMEngineerIdle_ResetProperties(int actor)
{
	ForgetRangeRepair(actor);
	m_hBuildingToGrab[actor] = INVALID_ENT_REFERENCE;
	g_bGoingToGrabBuilding[actor] = false;
	m_ctRecomputePathMvMEngiIdle[actor] = -1.0;
	m_ctSentrySafe[actor] = -1.0;
	m_ctSentryCooldown[actor] = -1.0;
	m_ctDispenserSafe[actor] = -1.0;
	m_ctDispenserCooldown[actor] = -1.0;
	m_ctFindNestHint[actor] = -1.0;
	m_ctAdvanceNestSpot[actor] = -1.0;
	g_arrPluginBot[actor].bPathing = true;
}

stock bool IsSentrySafe(int sentry)
{
	if (sentry == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (TF2_IsBuilding(sentry))
	{
		return false;
	}
	if (BaseEntity_GetHealth(sentry) < TF2Util_GetEntityMaxHealth(sentry))
	{
		return false;
	}
	if (!TF2_IsMiniBuilding(sentry) && (TF2_GetUpgradeLevel(sentry) < 3))
	{
		return false;
	}
	return GetEntProp(sentry, Prop_Send, "m_iAmmoShells") > 50;
}

stock bool CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(int actor)
{
	if (m_aNestArea[actor] == NULL_AREA)
	{
		return false;
	}
	int obj = GetObjectOfType(actor, TFObject_Sentry);
	if (obj == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (m_ctAdvanceNestSpot[actor] <= 0.0)
	{
		m_ctAdvanceNestSpot[actor] = GetGameTime() + 5.0;
		return false;
	}
	if (BaseEntity_GetHealth(obj) < TF2Util_GetEntityMaxHealth(obj))
	{
		m_ctAdvanceNestSpot[actor] = GetGameTime() + 5.0;
		return false;
	}
	if (GetGameTime() > m_ctAdvanceNestSpot[actor])
	{
		m_ctAdvanceNestSpot[actor] = -1.0;
	}
	BombInfo_t bombinfo;
	bool found = GetBombInfo(bombinfo);
	if (!found)
	{
		return false;
	}
	float flBombTargetDistance = GetTravelDistanceToBombTarget(m_aNestArea[actor]);
	if (flBombTargetDistance <= 1000.0)
	{
		return false;
	}
	bool bigger = flBombTargetDistance > bombinfo.flMaxBattleFront;
	return bigger;
}

stock void EngineerNestRelocation_OnWaveComplete()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_aNestAreaRelocate[i] = NULL_AREA;
	}
	EngineerNestRelocation_StopEvaluating();
	if (!redbots_manager_engineer_nest_relocate.BoolValue)
	{
		return;
	}
	m_iNestRelocateEvalNext = 1;
	m_hNestRelocateEvalTimer = CreateTimer(NEST_RELOCATE_EVAL_INTERVAL, Timer_EvaluateNestRelocation, _, TIMER_REPEAT);
}

stock void EngineerNestRelocation_StopEvaluating()
{
	StopNestRelocateEval();
}

stock void StopNestRelocateEval()
{
	m_iNestRelocateEvalNext = 1;
	if (m_hNestRelocateEvalTimer != null)
	{
		KillTimer(m_hNestRelocateEvalTimer);
		m_hNestRelocateEvalTimer = null;
	}
}

public Action Timer_EvaluateNestRelocation(Handle timer)
{
	if (m_iNestRelocateEvalNext > MaxClients)
	{
		m_hNestRelocateEvalTimer = null;
		return Plugin_Stop;
	}
	int client = m_iNestRelocateEvalNext;
	m_iNestRelocateEvalNext++;
	if (!ShouldEvaluateNestRelocation(client))
	{
		return Plugin_Continue;
	}
	CNavArea destination;
	bool move = ShouldRelocateNest(client, destination);
	if (!move)
	{
		return Plugin_Continue;
	}
	m_aNestAreaRelocate[client] = destination;
	if (redbots_manager_debug.BoolValue)
	{
		PrintToServer("EngineerNestRelocation: %N is moving nest", client);
	}
	return Plugin_Continue;
}

stock bool ShouldEvaluateNestRelocation(int client)
{
	if (!IsClientInGame(client) || !g_bIsDefenderBot[client] || !IsPlayerAlive(client))
	{
		return false;
	}
	if (TF2_GetPlayerClass(client) != TFClass_Engineer)
	{
		return false;
	}
	if (TF2_IsCarryingObject(client))
	{
		return false;
	}
	int sentry = GetObjectOfType(client, TFObject_Sentry);
	return !((sentry != INVALID_ENT_REFERENCE) && TF2_IsBuilding(sentry));
}

stock void EngineerNestRelocation_ResetAll()
{
	StopNestRelocateEval();
	for (int i = 1; i <= MaxClients; i++)
	{
		m_aNestAreaRelocate[i] = NULL_AREA;
		m_aNestAreaBeforeRelocate[i] = NULL_AREA;
		m_ctNestRelocateDeadline[i] = -1.0;
	}
}

public Action Command_DumpNest(int client, int args)
{
	ReplyToCommand(client, "%d nav areas on this map", TheNavAreas.Count);
	int building = -1;
	int standing = 0;
	for (;;)
	{
		building = FindEntityByClassname(building, "obj_*");
		if (building == -1)
		{
			break;
		}
		char class[512];
		GetEntityClassname(building, class, 512);
		int owner = GetEntPropEnt(building, Prop_Send, "m_hBuilder");
		float at[3];
		at = GetAbsOrigin(building);
		char whose[512];
		if ((owner > 0) && (owner <= MaxClients) && IsClientInGame(owner))
		{
			Format(whose, 512, "%N", owner);
		}
		else
		{
			Format(whose, 512, "nobody (orphan, owner index %d)", owner);
		}
		ReplyToCommand(client, "%s #%d at %.0f %.0f %.0f, built by %s", class, building, at[0], at[1], at[2], whose);
		standing++;
	}
	ReplyToCommand(client, "%d buildings standing", standing);
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || (TF2_GetPlayerClass(i) != TFClass_Engineer))
		{
			continue;
		}
		float nest[3];
		NestBuildPosition(m_aNestArea[i], nest);
		ReplyToCommand(client, "%N: nest %.0f %.0f %.0f", i, nest[0], nest[1], nest[2]);
		DumpBuilding(client, "sentry", GetObjectOfType(i, TFObject_Sentry));
		DumpBuilding(client, "dispenser", GetObjectOfType(i, TFObject_Dispenser));
		DumpBuilding(client, "entrance", GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Entrance));
		DumpBuilding(client, "exit", GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Exit));
		bool wants = ShouldBuildTeleporter(i);
		char lastResult[512];
		EngineerTeleporter_LastResult(i, lastResult, 64);
		ReplyToCommand(client, "  teleporter: round %d, sentry safe %s, gave up %s, wants %s%s, last \"%s\"", GameRules_GetRoundState(), (m_ctSentrySafe[i] > GetGameTime() ? "yes" : "no"), (EngineerTeleporter_HasGivenUp(i) ? "yes" : "no"), (wants ? "yes" : "no"), (ActionsManager.LookupEntityActionByName(i, "DefenderBuildTeleporter") != INVALID_ACTION ? ", building one now" : ""), lastResult);
		if (wants)
		{
			float spot[3];
			EngineerTeleporter_Spot(i, spot);
			ReplyToCommand(client, "  teleporter target: mode %d at %.0f %.0f %.0f", EngineerTeleporter_Mode(i), spot[0], spot[1], spot[2]);
		}
	}
	return Plugin_Handled;
}

stock void DumpBuilding(int client, const char[] what, int building)
{
	if (building == INVALID_ENT_REFERENCE)
	{
		ReplyToCommand(client, "  %s: none", what);
		return;
	}
	float origin[3];
	origin = GetAbsOrigin(building);
	ReplyToCommand(client, "  %s: level %d, %d of %d health, at %.0f %.0f %.0f%s", what, TF2_GetUpgradeLevel(building), BaseEntity_GetHealth(building), TF2Util_GetEntityMaxHealth(building), origin[0], origin[1], origin[2], (TF2_IsBuilding(building) ? ", still going up" : ""));
}

