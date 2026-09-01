#define SENTRY_WATCH_BOMB_RANGE	400.0

float m_ctSentrySafe[MAXPLAYERS + 1];

/* How much closer to the bomb the new ground has to be before the sentry is worth carrying

A sentry in a toolbox shoots nothing, and the walk there and back is most of a wave's opening. So
the candidate has to be meaningfully further forward, not merely different: without a margin, two
areas a few units apart trade places for ever and the engineer spends the wave carrying.

The cooldown is the second half of the same guard. Having moved, he holds what he has long enough
to have been worth moving. */
#define NEST_ADVANCE_MARGIN		600.0
#define NEST_ADVANCE_COOLDOWN	45.0
#define NEST_ADVANCE_RECHECK	10.0

static float m_ctAdvanceAgain[MAXPLAYERS + 1];

static bool IsWorthAdvancingTo(CNavArea held, CNavArea candidate)
{
	if (candidate == NULL_AREA || candidate == held)
		return false;

	if (held == NULL_AREA)
		return true;

	//Travel distance to where the bomb is going, so "forward" means along the route and not through a wall
	return GetTravelDistanceToBombTarget(view_as<CTFNavArea>(candidate))
		< GetTravelDistanceToBombTarget(view_as<CTFNavArea>(held)) - NEST_ADVANCE_MARGIN;
}

float m_ctSentryCooldown[MAXPLAYERS + 1];

float m_ctDispenserSafe[MAXPLAYERS + 1]; 
float m_ctDispenserCooldown[MAXPLAYERS + 1];

float m_ctFindNestHint[MAXPLAYERS + 1]; 
float m_ctAdvanceNestSpot[MAXPLAYERS + 1]; 

float m_ctRecomputePathMvMEngiIdle[MAXPLAYERS + 1];

bool g_bGoingToGrabBuilding[MAXPLAYERS + 1];
int m_hBuildingToGrab[MAXPLAYERS + 1];

//The nest an engineer was holding before a buster moved him off it, NULL_AREA when he is on it
CNavArea m_aNestAreaBeforeHaul[MAXPLAYERS + 1] = {NULL_AREA, ...};

//The nest an engineer is leaving for better ground, NULL_AREA when no relocation haul is running
CNavArea m_aNestAreaBeforeRelocate[MAXPLAYERS + 1] = {NULL_AREA, ...};

//When a relocation haul stops being worth finishing
float m_ctNestRelocateDeadline[MAXPLAYERS + 1];

/* How long an engineer gets to move a sentry to better ground before he puts it down where he is

The move is decided between waves and the wave can start while he is still walking. A sentry in a
toolbox when the robots arrive is worse than a badly placed level three, so the haul runs on a
clock rather than on a promise that it finishes in time */
#define NEST_RELOCATE_HAUL_TIME	20.0

/* How long an engineer may walk around holding a building before he puts it down
 *
 * The carry had no clock at all. He only tries to place while he is within seventy units of the
 * nest centre, so a centre he cannot reach, or a spot that keeps refusing the placement, leaves him
 * holding it for the rest of the mission. Reported from play on Coal Town, mid wave and between
 * waves both, and the mid wave one costs the team its sentry for the whole wave.
 *
 * Twenty five seconds is longer than any haul this mod starts on purpose and shorter than a wave.
 * Down where he stands beats carried, which is the same answer the relocation timeout already
 * gives: the ground under his feet is at worst ground he was already crossing. */
#define CARRY_GIVE_UP_TIME	25.0

static float m_ctCarryDeadline[MAXPLAYERS + 1];

float m_ctSentryUnderFire[MAXPLAYERS + 1];
int m_iSentryHealthLast[MAXPLAYERS + 1];

//How long a sentry counts as under fire after the last health it lost
#define SENTRY_UNDER_FIRE_TIME	3.0

BehaviorAction CTFBotMvMEngineerIdle()
{
	BehaviorAction action = ActionsManager.Create("DefenderEngineerIdle");
	
	action.OnStart = CTFBotMvMEngineerIdle_OnStart;
	action.Update = CTFBotMvMEngineerIdle_Update;
	action.OnEnd = CTFBotMvMEngineerIdle_OnEnd;
	action.OnMoveToSuccess = CTFBotMvMEngineerIdle_OnMoveToSuccess;
	
	return action;
}

static Action CTFBotMvMEngineerIdle_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	
	//A fresh engineer holds no ground yet, so there is nowhere for a buster to have moved him off
	m_aNestAreaBeforeHaul[actor] = NULL_AREA;
	m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
	m_ctNestRelocateDeadline[actor] = -1.0;
	m_iSentryHealthLast[actor] = 0;
	
	CTFBotMvMEngineerIdle_ResetProperties(actor);
	
	return action.Continue();
}

/* An engineer with no sentry who is not building one, said out loud every few seconds
 *
 * Measured on Coaltown: wave four ran for eight minutes and the engineer had a sentry standing for
 * fifteen percent of it, with one gap of three hundred and eighty five seconds. No build action
 * failed in that time, because no build action ran. He was inside this update the whole while, so
 * one of its early returns owns the wave and there is no way to tell which from outside.
 *
 * Throttled per engineer, and only while he has nothing standing, so a working nest is silent. */
#define ENGINEER_STALL_REPORT	10.0

static float m_ctEngineerStallReport[MAXPLAYERS + 1];

static void ReportEngineerStall(int actor, const char[] where)
{
	if (m_ctEngineerStallReport[actor] > GetGameTime())
		return;
	
	m_ctEngineerStallReport[actor] = GetGameTime() + ENGINEER_STALL_REPORT;
	
	PrintToServer("[defenderbots] engineer %N has no sentry at %.1f: %s (nest %s, carrying %s, grabbing %s)",
		actor, GetGameTime(), where,
		m_aNestArea[actor] == NULL_AREA ? "none" : "held",
		TF2_IsCarryingObject(actor) ? "yes" : "no",
		g_bGoingToGrabBuilding[actor] ? "yes" : "no");
}

static Action CTFBotMvMEngineerIdle_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	/* Counting what is in his hands, for both
	
	A carried building is not a reason to build a second one, and answering that question wrong is
	what a play-test filmed: the engineer stood holding a sentry, scrolling through his weapons and
	opening and cancelling the build menu, over and over. Two paths were undoing each other every
	frame. The carry logic equips the wrench and presses fire to put it down; the gate below saw no
	sentry, suspended for the build action, and that equips the toolbox and reopens the menu. */
	int sentry    = HasObjectOfType(actor, TFObject_Sentry);
	int dispenser = HasObjectOfType(actor, TFObject_Dispenser);
	
	//Standing, for the question of whether the nest is up: one in his hands is not defending anything
	int sentryStanding = GetObjectOfType(actor, TFObject_Sentry);
	
	bool stalled = sentryStanding == INVALID_ENT_REFERENCE && GameRules_GetRoundState() == RoundState_RoundRunning;
	
	bool bShouldAdvance = CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(actor);

	/* A buster is walking at the nest, so the nest stops being where the engineer wants to be
	Handled before the advance below and before anything else this action does, because the whole
	value of it is spending the walk the buster still has left. It reuses the carry machinery that
	advancing already uses: the goal area is somewhere out of the blast rather than a better nest,
	and the engineer picks the sentry up and walks it there the same way */
	int buster = -1;

	if (!g_bGoingToGrabBuilding[actor] && ShouldHaulFromSentryBuster(actor, sentry, buster))
	{
		CNavArea retreat = PickBusterRetreatArea(sentry, buster);

		if (retreat != NULL_AREA)
		{
			if (redbots_manager_debug_actions.BoolValue)
				PrintToServer("CTFBotMvMEngineerIdle_Update: HAUL FROM BUSTER");

			BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_INCOMING);

			/* The nest this engineer is leaving, so that it goes back to it
			Without this the spot the sentry was carried to becomes the nest for the rest of the
			wave, and a spot chosen for being far from one robot is not a spot to hold ground from */
			m_aNestAreaBeforeHaul[actor] = m_aNestArea[actor];

			CTFBotMvMEngineerIdle_ResetProperties(actor);

			m_aNestArea[actor] = retreat;

			g_bGoingToGrabBuilding[actor] = true;
			m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);

			g_arrPluginBot[actor].SetPathGoalEntity(sentry);

			return action.Continue();
		}
	}

	/* The buster is gone and the sentry is standing somewhere it was carried to, not somewhere
	chosen to shoot from. Going back is the same walk in the other direction */
	if (m_aNestAreaBeforeHaul[actor] != NULL_AREA && buster == -1 && !g_bGoingToGrabBuilding[actor] && sentry != INVALID_ENT_REFERENCE)
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

	/* The between-waves answer says this nest moved and the buildings are still standing on the old
	ground, which is what happens when nothing tore them down at the upgrade station. Same carry as
	the buster haul above, with better ground as the goal rather than open ground */
	if (m_aNestAreaRelocate[actor] != NULL_AREA
	&& m_aNestAreaBeforeHaul[actor] == NULL_AREA
	&& !g_bGoingToGrabBuilding[actor]
	&& !TF2_IsCarryingObject(actor)
	&& (sentry == INVALID_ENT_REFERENCE || !TF2_IsBuilding(sentry)))
	{
		CNavArea destination = m_aNestAreaRelocate[actor];
		
		//One move per between-waves period, whether or not there is anything to carry to it
		m_aNestAreaRelocate[actor] = NULL_AREA;
		
		if (sentry == INVALID_ENT_REFERENCE)
		{
			//Nothing to carry, so the new ground is simply where the next sentry goes up
			m_aNestArea[actor] = destination;
		}
		else
		{
			if (redbots_manager_debug_actions.BoolValue)
				PrintToServer("CTFBotMvMEngineerIdle_Update: RELOCATE NEST");
			
			m_aNestAreaBeforeRelocate[actor] = m_aNestArea[actor];
			
			CTFBotMvMEngineerIdle_ResetProperties(actor);
			
			m_ctNestRelocateDeadline[actor] = GetGameTime() + NEST_RELOCATE_HAUL_TIME;
			m_aNestArea[actor] = destination;
			
			g_bGoingToGrabBuilding[actor] = true;
			m_hBuildingToGrab[actor] = EntIndexToEntRef(sentry);
			
			g_arrPluginBot[actor].SetPathGoalEntity(sentry);
			
			/* The dispenser stays behind on ground nobody holds any more
			Only one building can be carried at a time and the sentry is the one worth the walk, so
			the dispenser is spent rather than dragged: the idle loop below builds another one at the
			new nest for 100 metal as soon as the sentry is safe */
			DetonateObjectOfType(actor, TFObject_Dispenser);
			
			return action.Continue();
		}
	}
	
	if (bShouldAdvance && !g_bGoingToGrabBuilding[actor])
	{
		/* Move, but only to ground that is actually further forward, and only once in a while

		Reported from play on Coaltown: the engineer on the building at the right picks the sentry
		up the moment the wave starts and keeps picking it up, and puts it down where it can see
		nothing.

		Both halves of that are this block. It re-scored the nest on every frame the advance
		condition held and moved to whatever came back, and what came back is not required to be
		any further forward than where he already was: PickBuildArea answers with the best area it
		can find, and if that is the one he is standing on, or another one equally far behind the
		front, the condition is still true next frame and he picks the sentry up again.

		So the candidate has to beat the ground he holds by a clear margin before he commits to
		carrying anything, and having moved, he leaves it alone for a while. A sentry in a toolbox
		shoots nothing, so a move has to buy more than it costs. */
		CNavArea candidate = PickBuildArea(actor);

		if (m_ctAdvanceAgain[actor] > GetGameTime() || !IsWorthAdvancingTo(m_aNestArea[actor], candidate))
		{
			//Nothing better within reach, so stop asking for a bit and hold what he has
			m_ctAdvanceNestSpot[actor] = GetGameTime() + NEST_ADVANCE_RECHECK;
		}
		else
		{
			if (redbots_manager_debug_actions.BoolValue)
				PrintToServer("CTFBotMvMEngineerIdle_Update: ADVANCE");

			//RIGHT NOW
			CTFBotMvMEngineerIdle_ResetProperties(actor);

			m_aNestArea[actor] = candidate;
			m_ctAdvanceAgain[actor] = GetGameTime() + NEST_ADVANCE_COOLDOWN;

			if (sentry != INVALID_ENT_REFERENCE && m_aNestArea[actor] != NULL_AREA)
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
	
	/* The clock ran out on a relocation, which means the wave started while he was still walking

	Down where he stands beats carried into the middle of a wave: the ground under his feet is at
	worst ground he was already crossing, and the alternative is a nest that exists in a toolbox.
	Before he picks the sentry up there is nothing to put down and the old nest is still a nest */
	if (m_aNestAreaBeforeRelocate[actor] != NULL_AREA && m_ctNestRelocateDeadline[actor] > 0.0 && GetGameTime() > m_ctNestRelocateDeadline[actor])
	{
		m_ctNestRelocateDeadline[actor] = -1.0;
		
		if (TF2_IsCarryingObject(actor))
		{
			CNavArea here = TheNavMesh.GetNearestNavArea(GetAbsOrigin(actor), false, 500.0, false, true, TEAM_ANY);
			
			if (here != NULL_AREA)
				m_aNestArea[actor] = here;
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
		
		/* The clock on the carry, started when he first has the thing in his hands
		
		Placing it needs him within seventy units of the nest centre, and nothing here says what to
		do when he never gets there. */
		if (!TF2_IsCarryingObject(actor))
		{
			m_ctCarryDeadline[actor] = 0.0;
		}
		else if (m_ctCarryDeadline[actor] <= 0.0)
		{
			m_ctCarryDeadline[actor] = GetGameTime() + CARRY_GIVE_UP_TIME;
		}
		else if (GetGameTime() > m_ctCarryDeadline[actor])
		{
			CNavArea here = TheNavMesh.GetNearestNavArea(GetAbsOrigin(actor), false, 500.0, false, true, TEAM_ANY);
			
			m_ctCarryDeadline[actor] = 0.0;
			
			LogBuildFailure(actor, "carry", "held it too long, putting it down here");
			
			//The nest is where he is now, so the placing branch below takes it the moment it runs
			if (here != NULL_AREA)
				m_aNestArea[actor] = here;
		}
		
		if (building == INVALID_ENT_REFERENCE)
		{
			g_bGoingToGrabBuilding[actor] = false;
			m_hBuildingToGrab[actor] = INVALID_ENT_REFERENCE;
			m_aNestAreaBeforeRelocate[actor] = NULL_AREA;
			m_ctNestRelocateDeadline[actor] = -1.0;
			
			if (redbots_manager_debug_actions.BoolValue)
				PrintToServer("CTFBotMvMEngineerIdle_Update: g_bGoingToGrabBuilding : building %i | m_aNestArea %x", building, m_aNestArea[actor]);
			
			DetonateObjectOfType(actor, TFObject_Sentry);
			DetonateObjectOfType(actor, TFObject_Dispenser);
			
			g_arrPluginBot[actor].bPathing = false;
			
			if (stalled)
				ReportEngineerStall(actor, "the building he was fetching is gone");
			
			return action.Continue();
		}
		
		UpdateLookAroundForEnemies(actor, false);
		
		if (!TF2_IsCarryingObject(actor))
		{
			float flDistanceToBuilding = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(building));
			
			if (flDistanceToBuilding < 90.0)
			{
				EquipWeaponSlot(actor, TFWeaponSlot_Melee);
				
				AimHeadTowards(myBody, WorldSpaceCenter(building), CRITICAL, 1.0, _, "Grab building");
				VS_PressAltFireButton(actor);
			}
		}
		else
		{
			if (m_aNestArea[actor] != NULL_AREA)
			{
				float center[3]; m_aNestArea[actor].GetCenter(center);
				g_arrPluginBot[actor].SetPathGoalVector(center);
				
				float flDistanceToGoal = GetVectorDistance(GetAbsOrigin(actor), center);
				
				if (flDistanceToGoal < 200.0)
				{
					//Crouch when closer than 200 hu
					if (!myLoco.IsStuck())
					{
						g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
					}
					
					if (flDistanceToGoal < 70.0)
					{
						//Try placing building when closer than 70 hu
						int objBeingBuilt = TF2_GetCarriedObject(actor);
						
						if (objBeingBuilt == -1)
							return action.Continue();
						
						bool m_bPlacementOK = IsPlacementOK(objBeingBuilt);
						
						VS_PressFireButton(actor);
						
						if (!m_bPlacementOK && myBody.IsHeadAimingOnTarget() && myBody.GetHeadSteadyDuration() > 0.6)
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
	
	if ((m_aNestArea[actor] == NULL_AREA || bShouldAdvance) || sentry == INVALID_ENT_REFERENCE)
	{
		//HasStarted && !IsElapsed
		if (m_ctFindNestHint[actor] > 0.0 && m_ctFindNestHint[actor] > GetGameTime())
		{
			return action.Continue();
		}
		
		//Start
		m_ctFindNestHint[actor] = GetGameTime() + (GetRandomFloat(1.0, 2.0));
		
		m_aNestArea[actor] = PickBuildArea(actor);
	}
	
	if (bShouldAdvance)
	{
		if (stalled)
			ReportEngineerStall(actor, "advancing the nest");
		
		return action.Continue();
	}
	
	UpdateSentryUnderFire(actor, sentry);

	if (sentry != -1)
	{
		if ((m_ctSentrySafe[actor] > GetGameTime() || m_ctSentryUnderFire[actor] > GetGameTime()) && !g_bGoingToGrabBuilding[actor])
		{
			int mySecondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);

			if (mySecondary != -1 && TF2Util_GetWeaponID(mySecondary) == TF_WEAPON_LASER_POINTER && myNextbot.IsRangeLessThan(sentry, 180.0))
			{
				CKnownEntity threat = myNextbot.GetVisionInterface().GetPrimaryKnownThreat(false);

				if (threat)
				{
					int iThreat = threat.GetEntity();

					/* Two reasons to hold the wrangler, and the shield is the one that was missing

					Out of range, the wrangler is the only way the sentry reaches the threat at all,
					which is what this did and all it did. Under fire, it is worth holding for the
					shield alone: two thirds of the damage aimed at the sentry stops there, and a
					sentry that survives the giant is worth more than the seconds of aim it costs.

					Both want the same thing of the bot, so both run the same code: point it at the
					threat and hold the buttons */
					bool defending = m_ctSentryUnderFire[actor] > GetGameTime();

					if ((defending || GetVectorDistance(GetAbsOrigin(sentry), GetAbsOrigin(iThreat)) > SENTRY_MAX_RANGE) && IsLineOfFireClearEntity(actor, GetEyePosition(actor), iThreat))
					{
						AimHeadTowards(myBody, WorldSpaceCenter(iThreat), MANDATORY, 0.1, _, "Aiming!");
						TF2Util_SetPlayerActiveWeapon(actor, mySecondary);
						
						if (myBody.IsHeadAimingOnTarget() && GetEntProp(sentry, Prop_Send, "m_bPlayerControlled"))
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
	
	if (m_aNestArea[actor] == NULL_AREA && stalled)
		ReportEngineerStall(actor, "no nest area to build on");
	
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
			/* do not have a sentry; retreat for a few seconds if we had a
			 * sentry before this; then build a new sentry */
			if (m_ctSentryCooldown[actor] >= GetGameTime())
				ReportEngineerStall(actor, "waiting out the rebuild cooldown");
			
			if (m_ctSentryCooldown[actor] < GetGameTime()) 
			{
				m_ctSentryCooldown[actor] = GetGameTime() + 3.0;
				
				return action.SuspendFor(CTFBotMvMEngineerBuildSentrygun(), "No sentry - building a new one");
			}
		}
		
		//Don't build a dispenser if we don't have a sentry...
		if (sentry != INVALID_ENT_REFERENCE)
		{
			if (dispenser != INVALID_ENT_REFERENCE)
			{
				//sentry is not safe.
				if (m_ctSentrySafe[actor] < GetGameTime())
				{
					m_ctDispenserCooldown[actor] = GetGameTime() + 3.0;
				}
			}
			else 
			{
				/* do not have a dispenser; retreat for a few seconds if we had a
				 * dispenser before this; then build a new dispenser */
				if (m_ctDispenserCooldown[actor] < GetGameTime() && m_ctSentrySafe[actor] > GetGameTime())
				{
					m_ctDispenserCooldown[actor] = GetGameTime() + 3.0;
					
					return action.SuspendFor(CTFBotMvMEngineerBuildDispenser(), "Sentry safe, No dispenser - building one");
				}
			}
		}
	}
	
	/* The nest is standing and the wave has not started, so there is time for a teleporter
	Nothing below this point is skipped by it: the action gives up the moment the wave starts or
	the sentry stops standing, and the engineer goes straight back to the nest */
	if (m_ctSentrySafe[actor] > GetGameTime() && !g_bGoingToGrabBuilding[actor] && ShouldBuildTeleporter(actor))
		return action.SuspendFor(CTFBotMvMEngineerBuildTeleporter(), "Nest is up, building a teleporter");
	
	//A second gun beside the first, put there on purpose rather than dropped wherever he was facing
	if (m_ctSentrySafe[actor] > GetGameTime() && !g_bGoingToGrabBuilding[actor] && ShouldBuildDisposable(actor))
		return action.SuspendFor(CTFBotMvMEngineerBuildDisposable(), "Nest is up, standing a mini beside it");
	
	/* The dispenser is only a job once the sentry is not one
	
	This branch comes first and returns, so an engineer whose dispenser was a level two stood at it
	swinging a wrench while the sentry twenty feet away was being shot to pieces. The sentry is the
	nest; the dispenser is what feeds it. Reported as the engineer not repairing his buildings,
	which he was doing, just never the one being destroyed. */
	bool sentryWantsMetal = sentry != INVALID_ENT_REFERENCE && SentryNeedsMetal(sentry);
	
	if (dispenser != INVALID_ENT_REFERENCE && !sentryWantsMetal && m_ctSentrySafe[actor] > GetGameTime())
	{
		if (TF2_GetUpgradeLevel(dispenser) < 3 || BaseEntity_GetHealth(dispenser) < TF2Util_GetEntityMaxHealth(dispenser))
		{
			float dist = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(dispenser));
			
			if (m_ctRecomputePathMvMEngiIdle[actor] < GetGameTime()) 
			{
				m_ctRecomputePathMvMEngiIdle[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
				
				float dir[3];
				SubtractVectors(GetAbsAngles(dispenser), GetAbsOrigin(actor), dir);
				NormalizeVector(dir, dir);
				
				float goal[3]; goal = GetAbsOrigin(dispenser);
				goal[0] -= (50.0 * dir[0]);
				goal[1] -= (50.0 * dir[1]);
				goal[2] -= (50.0 * dir[2]);
				
				if (IsPathToVectorPossible(actor, goal, _))
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
				
				AimHeadTowards(myBody, WorldSpaceCenter(dispenser), CRITICAL, 1.0, _, "Work on my Dispenser");
				VS_PressFireButton(actor);
			}
			
			return action.Continue();
		}
	}
	
	if (sentry != INVALID_ENT_REFERENCE) 
	{
		float dist = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(sentry));
		
		/* A finished sentry is not a job
		The wrench does nothing to a level three at full health and full shells, and the engineer
		swinging it is an engineer not shooting at the robots walking into his nest */
		if (!SentryNeedsMetal(sentry))
		{
			EquipWeaponSlot(actor, TFWeaponSlot_Primary);
			
			UpdateLookAroundForEnemies(actor, true);
		}
		else if (CanRepairFromRange(actor, sentry, dist))
		{
			//The Rescue Ranger repairs from behind cover, which is the whole reason to carry it
			EquipWeaponSlot(actor, TFWeaponSlot_Primary);
			
			AimHeadTowards(myBody, WorldSpaceCenter(sentry), CRITICAL, 1.0, _, "Repair my Sentry from here");
			
			if (myBody.IsHeadAimingOnTarget())
				VS_PressFireButton(actor);
			
			g_arrPluginBot[actor].bPathing = false;
			
			return action.Continue();
		}
		
		if (m_ctRecomputePathMvMEngiIdle[actor] < GetGameTime()) 
		{
			m_ctRecomputePathMvMEngiIdle[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
			
			float vTurretAngles[3]; GetTurretAngles(sentry, vTurretAngles);
			float dir[3];
			GetAngleVectors(vTurretAngles, dir, NULL_VECTOR, NULL_VECTOR);
			
			float goal[3]; goal = GetAbsOrigin(sentry);
			goal[0] -= (50.0 * dir[0]);
			goal[1] -= (50.0 * dir[1]);
			goal[2] -= (50.0 * dir[2]);
			
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
		
		if (dist < 90.0 && SentryNeedsMetal(sentry)) 
		{
			if (!myLoco.IsStuck())
			{
				g_arrExtraButtons[actor].PressButtons(IN_DUCK, 0.1);
			}
			
			EquipWeaponSlot(actor, TFWeaponSlot_Melee);
			
			UpdateLookAroundForEnemies(actor, false);
			
			AimHeadTowards(myBody, WorldSpaceCenter(sentry), CRITICAL, 1.0, _, "Work on my Sentry");
			VS_PressFireButton(actor);
		}
	}
	
	return action.Continue();
}

/* Whether the sentry is being shot at, from the only thing that says so without a damage hook

Health that went down since the last frame. A repair puts it back up and does not clear the
timer, which is the point: an engineer wrenching a sentry that keeps losing health is an engineer
who should be holding the wrangler instead */
static void UpdateSentryUnderFire(int actor, int sentry)
{
	if (sentry == INVALID_ENT_REFERENCE)
	{
		m_iSentryHealthLast[actor] = 0;
		return;
	}

	int health = BaseEntity_GetHealth(sentry);

	if (m_iSentryHealthLast[actor] > 0 && health < m_iSentryHealthLast[actor])
		m_ctSentryUnderFire[actor] = GetGameTime() + SENTRY_UNDER_FIRE_TIME;

	m_iSentryHealthLast[actor] = health;
}

/* Whether to pick the sentry up and walk it away from a buster, and which buster

Only worth doing while the buster still has ground to cover. Inside flee range the engineer is
better off running than carrying, which is what CTFBotEvadeBuster does with him, and a buster
already detonating cannot be walked away from at all.

A mini sentry is not carried. It costs 100 metal and two seconds, so a Gunslinger engineer lets
the buster have it and builds another one behind the blast */
static bool ShouldHaulFromSentryBuster(int actor, int sentry, int &buster)
{
	buster = -1;

	if (sentry == INVALID_ENT_REFERENCE)
		return false;

	if (GameRules_GetRoundState() != RoundState_RoundRunning)
		return false;

	if (TF2_IsCarryingObject(actor) || TF2_IsBuilding(sentry))
		return false;

	buster = FindSentryBusterNear(GetAbsOrigin(sentry), GetPlayerEnemyTeam(actor), BUSTER_HAUL_RANGE);

	if (buster == -1)
		return false;

	if (TF2_IsMiniBuilding(sentry))
		return false;

	//Close enough that carrying is slower than the buster is
	if (GetVectorDistance(GetAbsOrigin(sentry), WorldSpaceCenter(buster)) < BUSTER_FLEE_RANGE)
		return false;

	return true;
}

/* Whether there is anything the wrench can still do for this sentry

A mini sentry cannot be upgraded and is rebuilt rather than nursed, so for a Gunslinger this is
only ever about damage taken. Shells are counted because a sentry out of ammo is a sentry that
does nothing while it looks perfectly healthy */
bool SentryNeedsMetal(int sentry)
{
	if (TF2_IsBuilding(sentry))
		return true;
	
	if (BaseEntity_GetHealth(sentry) < TF2Util_GetEntityMaxHealth(sentry))
		return true;
	
	if (!TF2_IsMiniBuilding(sentry) && TF2_GetUpgradeLevel(sentry) < 3)
		return true;
	
	return GetEntProp(sentry, Prop_Send, "m_iAmmoShells") < 100;
}

/* How often a bolt was fired at a sentry that then failed to gain any health
 *
 * A player reported an engineer stood firing Rescue Ranger bolts into a wall with the sentry behind
 * it. Reproducing that here found nothing, and the reason is worth keeping: across four waves the
 * state this can happen in, a damaged sentry with its engineer standing two hundred units or more
 * away, occurred in zero samples out of a hundred and thirty seven. The sentry is at full health or
 * it is dead, and five second sampling cannot see a rare state at all.
 *
 * So this counts the event rather than sampling the state. A counter accumulates, and sampling a
 * counter loses nothing however rare the thing is: one stall in an hour still shows up as a one.
 *
 * It only counts. Deciding what the engineer should do about it needs to know how often it happens
 * and that is what this is for.
 */
#define RANGE_REPAIR_PATIENCE	3.0

static int m_iRangeRepairHealth[MAXPLAYERS + 1];
static float m_ctRangeRepairSince[MAXPLAYERS + 1];
static int m_iRangeRepairStalls[MAXPLAYERS + 1];

int RangeRepairStallsOf(int client)
{
	return m_iRangeRepairStalls[client];
}

static void NoteRangeRepair(int actor, int sentry)
{
	int health = BaseEntity_GetHealth(sentry);
	float now = GetGameTime();

	//Health went up, or this is the first bolt: either way the clock starts here
	if (health > m_iRangeRepairHealth[actor] || m_ctRangeRepairSince[actor] <= 0.0)
	{
		m_iRangeRepairHealth[actor] = health;
		m_ctRangeRepairSince[actor] = now;

		return;
	}

	m_iRangeRepairHealth[actor] = health;

	if (now - m_ctRangeRepairSince[actor] < RANGE_REPAIR_PATIENCE)
		return;

	//Counted once per stall rather than once per frame of one
	m_ctRangeRepairSince[actor] = now;
	m_iRangeRepairStalls[actor]++;

	LogBuildFailure(actor, "repairing at range", "three seconds of bolts and the sentry gained nothing");
}

void ForgetRangeRepair(int actor)
{
	m_iRangeRepairHealth[actor] = 0;
	m_ctRangeRepairSince[actor] = 0.0;
}

//A Rescue Ranger bolt repairs at range, so its engineer does not walk into the open to hold a nest
bool CanRepairFromRange(int actor, int sentry, float dist)
{
	if (!TF2_IsRescueRangerEquipped(actor))
		return false;
	
	/* A bolt repairs a building and does not reload one
	
	Only a hit with the wrench puts shells back in a sentry. A sentry at full health with an empty
	magazine still answers yes to SentryNeedsMetal, so the engineer stood at range firing bolts into
	something that was already whole and never gained a shell. Reported from play as Rescue Ranger
	engineers refusing to reload, and measured on Coal Town before the fix: 21 of 126 samples of a
	full health sentry had it under fifty shells, and some at none.
	
	So range repair is for damage and nothing else. Ammo is a walk. */
	if (BaseEntity_GetHealth(sentry) >= TF2Util_GetEntityMaxHealth(sentry))
		return false;
	
	//Close enough to swing at, and the wrench repairs faster
	if (dist < 200.0)
		return false;
	
	if (dist > SENTRY_MAX_RANGE)
		return false;
	
	if (GetEntProp(actor, Prop_Data, "m_iAmmo", _, 3) < 30)
		return false;
	
	if (!IsLineOfFireClearEntity(actor, GetEyePosition(actor), sentry))
	{
		ForgetRangeRepair(actor);
		
		return false;
	}
	
	//Watched, not acted on. What to do about a stall is a separate argument and needs this first
	NoteRangeRepair(actor, sentry);
	
	return true;
}

static void CTFBotMvMEngineerIdle_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	//NOTE: engineer should only truly leave this behavior when he dies, it should otherwise be impossible
	g_arrPluginBot[actor].bPathing = false;
}

static Action CTFBotMvMEngineerIdle_OnMoveToSuccess(BehaviorAction action, int actor, any path, ActionDesiredResult result)
{
	//Because of our constant pathing, we are not stuck once we arrive to our desired position
	CBaseNPC_GetNextBotOfEntity(actor).GetLocomotionInterface().ClearStuckStatus("Arrived at goal");
	
	return action.TryContinue();
}

static void CTFBotMvMEngineerIdle_ResetProperties(int actor)
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

/* What each engineer has actually got standing, and where

An engineer who never finishes a building looks the same from outside as one who never started,
and a teleporter half of a pair looks the same as none. This says which, and where each piece
ended up, so a spot that refuses everything can be walked to with sm_dump_spot. */
public Action Command_DumpNest(int client, int args)
{
	/* How many areas a nest decision walks, which is the size of everything else here

	PickBuildArea and GetBombInfo both walk the whole mesh, so this number is the unit that any
	"why did the frame take that long" answer is counted in. */
	ReplyToCommand(client, "%d nav areas on this map", TheNavAreas.Count);

	/* Every building standing, and who the game says owns it

	Asked for because a play-test found two dispensers with one engineer on the team, which is a
	thing the game does not let a player do: an engineer placing a second one has his first taken
	down for him. So one of them belongs to somebody else, or to nobody, and the per-engineer
	listing below cannot show either. This walks the entities instead of the players. */
	int building = -1;
	int standing = 0;

	while ((building = FindEntityByClassname(building, "obj_*")) != -1)
	{
		char class[64]; GetEntityClassname(building, class, sizeof(class));

		int owner = GetEntPropEnt(building, Prop_Send, "m_hBuilder");
		float at[3]; at = GetAbsOrigin(building);

		char whose[64];
		
		if (owner > 0 && owner <= MaxClients && IsClientInGame(owner))
			Format(whose, sizeof(whose), "%N", owner);
		else
			Format(whose, sizeof(whose), "nobody (orphan, owner index %d)", owner);
		
		ReplyToCommand(client, "%s #%d at %.0f %.0f %.0f, built by %s", class, building, at[0], at[1], at[2], whose);

		standing++;
	}

	ReplyToCommand(client, "%d buildings standing", standing);

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetPlayerClass(i) != TFClass_Engineer)
			continue;

		float nest[3]; NestBuildPosition(m_aNestArea[i], nest);

		ReplyToCommand(client, "%N: nest %.0f %.0f %.0f", i, nest[0], nest[1], nest[2]);

		DumpBuilding(client, "sentry", GetObjectOfType(i, TFObject_Sentry));
		DumpBuilding(client, "dispenser", GetObjectOfType(i, TFObject_Dispenser));
		DumpBuilding(client, "entrance", GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Entrance));
		DumpBuilding(client, "exit", GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Exit));

		//Asking moves the engineer's pending teleporter target, which the idle action recomputes anyway
		bool wants = ShouldBuildTeleporter(i);

		char lastResult[64]; EngineerTeleporter_LastResult(i, lastResult, sizeof(lastResult));

		ReplyToCommand(client, "  teleporter: round %d, sentry safe %s, gave up %s, wants %s%s, last \"%s\"",
			GameRules_GetRoundState(),
			m_ctSentrySafe[i] > GetGameTime() ? "yes" : "no",
			EngineerTeleporter_HasGivenUp(i) ? "yes" : "no",
			wants ? "yes" : "no",
			ActionsManager.LookupEntityActionByName(i, "DefenderBuildTeleporter") != INVALID_ACTION ? ", building one now" : "",
			lastResult);

		if (wants)
		{
			float spot[3]; EngineerTeleporter_Spot(i, spot);

			ReplyToCommand(client, "  teleporter target: mode %d at %.0f %.0f %.0f",
				EngineerTeleporter_Mode(i), spot[0], spot[1], spot[2]);
		}
	}

	return Plugin_Handled;
}

static void DumpBuilding(int client, const char[] what, int building)
{
	if (building == INVALID_ENT_REFERENCE)
	{
		ReplyToCommand(client, "  %s: none", what);

		return;
	}

	float origin[3]; origin = GetAbsOrigin(building);

	ReplyToCommand(client, "  %s: level %d, %d of %d health, at %.0f %.0f %.0f%s",
		what, TF2_GetUpgradeLevel(building), BaseEntity_GetHealth(building),
		TF2Util_GetEntityMaxHealth(building), origin[0], origin[1], origin[2],
		TF2_IsBuilding(building) ? ", still going up" : "");
}

/* Whether the sentry is finished and in no trouble, asked of the sentry rather than of a clock

m_ctSentrySafe is this answer with three seconds of life on it, and the three seconds are there so
that the flag survives a frame where the sentry is briefly not full. It is refreshed by the idle
action and only by the idle action.

That is fine for the idle action's own branches and wrong for anything the idle action suspends
into, because suspending stops its update running: the flag then expires three seconds later
whatever the sentry is actually doing. The dispenser build read it and ended itself with "Sentry
not safe" three seconds after starting, every time, on every map. It got a dispenser up only where
the walk and the placement both fitted inside those three seconds, which is why the dispenser was
standing for 13% of a wave on Mannworks and 45% on Mannhattan.

So the question is asked of the sentry where the answer has to be current, and the flag stays for
what it was for. */
bool IsSentrySafe(int sentry)
{
	if (sentry == INVALID_ENT_REFERENCE)
		return false;
	
	if (TF2_IsBuilding(sentry))
		return false;
	
	if (BaseEntity_GetHealth(sentry) < TF2Util_GetEntityMaxHealth(sentry))
		return false;
	
	if (!TF2_IsMiniBuilding(sentry) && TF2_GetUpgradeLevel(sentry) < 3)
		return false;
	
	return GetEntProp(sentry, Prop_Send, "m_iAmmoShells") > 50;
}

bool CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(int actor)
{
	if (m_aNestArea[actor] == NULL_AREA)
		return false;
	
	int obj = GetObjectOfType(actor, TFObject_Sentry);
	
	/* Nothing to advance, and saying otherwise stopped him building at all
	
	The idle action asks this first and returns without doing anything when the answer is yes,
	because advancing is the thing it is about to do. An engineer whose sentry has just been
	destroyed has nothing to move, so the answer has to be no: it was yes, and he stood in the
	middle of Bigrock for the rest of the wave with no sentry, no dispenser, and every branch that
	would have built one behind that return.
	
	It was expensive as well as wrong. Two frames of that is two calls to PickBuildArea, and
	PickBuildArea calls GetBombInfo, which walks every nav area on the map. Sixty-six times a
	second, twice, per engineer, for as long as he had no sentry. */
	if (obj == INVALID_ENT_REFERENCE)
		return false;
	
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
	
	//IsElapsed
	if (GetGameTime() > m_ctAdvanceNestSpot[actor])
	{
		m_ctAdvanceNestSpot[actor] = -1.0;
	}
	
	BombInfo_t bombinfo;
	
	if (!GetBombInfo(bombinfo)) 
	{
		return false;
	}
	
	float m_flBombTargetDistance = GetTravelDistanceToBombTarget(m_aNestArea[actor]);
	
	//No point in advancing now.
	if (m_flBombTargetDistance <= 1000.0)
	{
		return false;
	}
	
	bool bigger = (m_flBombTargetDistance > bombinfo.flMaxBattleFront);
	
	return bigger;
}
/* Ask every engineer whether the ground he holds is still the ground he wants, once per wave

Wave complete rather than wave start, because that is the last moment the old buildings are still
his to decide about: the upgrade session that follows tears them down, and the answer here is what
tells it whether to. It also gives him the whole shopping period to act on the answer instead of
carrying a sentry while the robots walk in.

An engineer with no nest, a dead one, or one whose sentry is still going up is left alone. There is
nothing to compare against in the first two cases, and in the third nobody yet knows what the
building is worth */
/* Deciding this costs a full nav mesh sweep per engineer

GetBombInfo walks every area, PickBuildArea walks every area and scores what survives, and the
sight score traces from each candidate to every sampled approach area. Doing that for six
engineers inside the wave_complete frame is how the server's watchdog gets tripped: it does not
care that the work is finite, only that the frame took seconds.

So one engineer per timer tick. The shopping period is a minute and the queue is at most the
server's player count, which is a rounding error against that. */
#define NEST_RELOCATE_EVAL_INTERVAL	0.1

static int m_iNestRelocateEvalNext;
static Handle m_hNestRelocateEvalTimer;

void EngineerNestRelocation_OnWaveComplete()
{
	for (int i = 1; i <= MaxClients; i++)
		m_aNestAreaRelocate[i] = NULL_AREA;
	
	StopNestRelocateEval();
	
	if (!redbots_manager_engineer_nest_relocate.BoolValue)
		return;
	
	m_iNestRelocateEvalNext = 1;
	m_hNestRelocateEvalTimer = CreateTimer(NEST_RELOCATE_EVAL_INTERVAL, Timer_EvaluateNestRelocation, _, TIMER_REPEAT);
}

//The wave started, or the round did: whatever the queue had left is about a bomb that has moved
void EngineerNestRelocation_StopEvaluating()
{
	StopNestRelocateEval();
}

static void StopNestRelocateEval()
{
	m_iNestRelocateEvalNext = 1;
	
	if (m_hNestRelocateEvalTimer != null)
	{
		KillTimer(m_hNestRelocateEvalTimer);
		m_hNestRelocateEvalTimer = null;
	}
}

static Action Timer_EvaluateNestRelocation(Handle timer)
{
	if (m_iNestRelocateEvalNext > MaxClients)
	{
		m_hNestRelocateEvalTimer = null;
		return Plugin_Stop;
	}
	
	int client = m_iNestRelocateEvalNext++;
	
	if (!ShouldEvaluateNestRelocation(client))
		return Plugin_Continue;
	
	CNavArea destination;
	
	if (!ShouldRelocateNest(client, destination))
		return Plugin_Continue;
	
	m_aNestAreaRelocate[client] = destination;
	
	if (redbots_manager_debug.BoolValue)
		PrintToServer("EngineerNestRelocation: %N is moving nest", client);
	
	return Plugin_Continue;
}

static bool ShouldEvaluateNestRelocation(int client)
{
	if (!IsClientInGame(client) || !g_bIsDefenderBot[client] || !IsPlayerAlive(client))
		return false;
	
	if (TF2_GetPlayerClass(client) != TFClass_Engineer)
		return false;
	
	if (TF2_IsCarryingObject(client))
		return false;
	
	int sentry = GetObjectOfType(client, TFObject_Sentry);
	
	return !(sentry != INVALID_ENT_REFERENCE && TF2_IsBuilding(sentry));
}

void EngineerNestRelocation_ResetAll()
{
	StopNestRelocateEval();
	
	for (int i = 1; i <= MaxClients; i++)
	{
		m_aNestAreaRelocate[i] = NULL_AREA;
		m_aNestAreaBeforeRelocate[i] = NULL_AREA;
		m_ctNestRelocateDeadline[i] = -1.0;
	}
}
