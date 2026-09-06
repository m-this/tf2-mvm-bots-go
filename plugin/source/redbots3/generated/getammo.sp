BehaviorAction CTFBotGetAmmo()
{
	BehaviorAction action = ActionsManager.Create("DefenderGetAmmo");

	action.OnStart = CTFBotGetAmmo_OnStart;
	action.Update = CTFBotGetAmmo_Update;
	action.OnEnd = CTFBotGetAmmo_OnEnd;
	action.ShouldHurry = CTFBotGetAmmo_ShouldHurry;
	action.ShouldAttack = CTFBotGetAmmo_ShouldAttack;

	return action;
}

#define AMMO_CANDIDATES_MAX (4)
#define AMMO_REPATH_FAILS_MAX (3)
#define AMMO_GIVEUP_TIME (3.0)

#define AMMO_ASK_INTERVAL (0.5)

#define HEALTH_CANDIDATES_MAX (64)
#define HEALTH_PATHS_MAX (4)

int m_iAmmoPack[65];
int m_arrAmmoCandidates[65][4];
int m_iAmmoCandidateCount[65];
int m_iAmmoCandidate[65];
int m_iAmmoRepathFails[65];
float m_ctAmmoAsk[65];
bool m_bAmmoPossible[65];
char g_strHealthAndAmmoEntities[][] =
{
	"func_regenerate",
	"item_ammopack*",
	"item_health*",
	"obj_dispenser",
	"tf_ammo_pack",
};

// OnStart ranks the packs in range and takes the nearest.
public Action CTFBotGetAmmo_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	// Nothing unless a debug convar is set, which is never on a real server
	DebugFaults_OnAmmoWalkStart(actor);
	ArrayList ammo = new ArrayList(2);
	ComputeHealthAndAmmoVectors(actor, ammo, tf_bot_ammo_search_range.FloatValue);
	m_iAmmoPack[actor] = -1;
	m_iAmmoCandidateCount[actor] = 0;
	m_iAmmoCandidate[actor] = 0;
	m_iAmmoRepathFails[actor] = 0;
	// Shortest travel first, so a failover walks outwards rather than anywhere
	while (m_iAmmoCandidateCount[actor] < AMMO_CANDIDATES_MAX)
	{
		int best = -1;
		float flSmallestDistance = 0.0;
		for (int i = 0; i < ammo.Length; i++)
		{
			int entity = ammo.Get(i, 0);
			if ((entity == -1) || !IsValidAmmo(entity))
			{
				continue;
			}
			float flDistance = view_as<float>(ammo.Get(i, 1));
			if ((best == -1) || (flDistance < flSmallestDistance))
			{
				best = i;
				flSmallestDistance = flDistance;
			}
		}
		if (best == -1)
		{
			break;
		}
		m_arrAmmoCandidates[actor][m_iAmmoCandidateCount[actor]] = ammo.Get(best, 0);
		m_iAmmoCandidateCount[actor]++;
		ammo.Set(best, -1, 0);
	}
	if (m_iAmmoCandidateCount[actor] > 0)
	{
		m_iAmmoPack[actor] = m_arrAmmoCandidates[actor][0];
		if (TF2_GetPlayerClass(actor) == TFClass_Engineer)
		{
			UpdateLookAroundForEnemies(actor, true);
		}
		BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_DISPENSERHERE);
		delete ammo;
		return action.Continue();
	}
	delete ammo;
	return action.Done("Could not find ammo");
}

// Update walks to the pack, and to the next one along when the route to this one
// stops existing.
public Action CTFBotGetAmmo_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidAmmo(m_iAmmoPack[actor]))
	{
		return action.Done("ammo is not valid");
	}
	if (IsAmmoFull(actor))
	{
		return action.Done("Ammo is full");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.9, 1.0);
		RepathToPos(actor, myBot, WorldSpaceCenter(m_iAmmoPack[actor]));
		if (Feature(FEATURE_AMMO_FAILOVER))
		{
			// The return value is the only thing that says the route failed. The length lies.
			if (!DebugFaults_RefuseAmmoPath(actor) && !PathFailedFor(actor))
			{
				m_iAmmoRepathFails[actor] = 0;
			}
			else
			{
				m_iAmmoRepathFails[actor]++;
				if (m_iAmmoRepathFails[actor] >= AMMO_REPATH_FAILS_MAX)
				{
					m_iAmmoRepathFails[actor] = 0;
					if (!NextAmmoCandidate(actor))
					{
						HoldOffAmmo(actor);
						return action.Done("No reachable ammo");
					}
					RepathToPos(actor, myBot, WorldSpaceCenter(m_iAmmoPack[actor]));
				}
			}
		}
	}
	m_pPath[actor].Update(myBot);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		EquipBestWeaponForThreat(actor, threat);
	}
	return action.Continue();
}

// OnEnd forgets the pack and the ranking behind it.
public void CTFBotGetAmmo_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iAmmoPack[actor] = -1;
	m_iAmmoCandidateCount[actor] = 0;
	m_iAmmoCandidate[actor] = 0;
	m_iAmmoRepathFails[actor] = 0;
}

// NextCandidate is the next pack he was ranked onto, skipping any taken while he
// walked. Bounded by the list.
stock bool NextAmmoCandidate(int actor)
{
	for (m_iAmmoCandidate[actor]++; m_iAmmoCandidate[actor] < m_iAmmoCandidateCount[actor]; m_iAmmoCandidate[actor]++)
	{
		int pack = m_arrAmmoCandidates[actor][m_iAmmoCandidate[actor]];
		if (!IsValidAmmo(pack))
		{
			continue;
		}
		m_iAmmoPack[actor] = pack;
		return true;
	}
	return false;
}

// ShouldHurry disables dodging, and keeps the minigun unspun after recently
// seeing threats.
public Action CTFBotGetAmmo_ShouldHurry(BehaviorAction action, INextBot nextbot, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	result = ANSWER_YES;
	return Plugin_Handled;
}

// ShouldAttack keeps a spy walking for ammo out of a fight it cannot win.
public Action CTFBotGetAmmo_ShouldAttack(BehaviorAction action, INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result)
{
	result = view_as<QueryResultType>(0);
	int me = action.Actor;
	if (TF2_GetPlayerClass(me) == TFClass_Spy)
	{
		int iThreat = knownEntity.GetEntity();
		if (BaseEntity_IsPlayer(iThreat) && (GetClientHealth(iThreat) > 360) && !TF2_IsCritBoosted(me))
		{
			// Don't attack if we can't possibly kill them with our revolver (360 from 6 shots with max damage)
			result = ANSWER_NO;
			return Plugin_Changed;
		}
		else
			if (GetNearestEnemyCount(me, 1000.0, false) > 1)
			{
				// There's too many enemies nearby, it'd be better to redisguise so they'll forget about us
				result = ANSWER_NO;
				return Plugin_Changed;
			}
	}
	result = ANSWER_UNDEFINED;
	return Plugin_Changed;
}

// IsValidAmmo says the entity is ammo the bot could still take.
stock bool IsValidAmmo(int pack)
{
	if (!IsValidEntity(pack))
	{
		return false;
	}
	if (!HasEntProp(pack, Prop_Send, "m_fEffects"))
	{
		return false;
	}
	// It has been taken.
	if (GetEntProp(pack, Prop_Send, "m_fEffects") != 0)
	{
		return false;
	}
	char class[512];
	GetEntityClassname(pack, class, 512);
	if ((StrContains(class, "tf_ammo_pack", false) == -1) && (StrContains(class, "item_ammo", false) == -1) && (StrContains(class, "obj_dispenser", false) == -1) && (StrContains(class, "func_regen", false) == -1))
	{
		return false;
	}
	// Can't use a disabled dispenser
	if ((StrContains(class, "obj_dispenser", false) != -1) && TF2_HasSapper(pack))
	{
		return false;
	}
	return true;
}

// HoldOff keeps the gate shut after a walk that ran out of reachable packs.
//
// The cache answers from a nav search that said yes, and the walk that followed
// said no. Without this the monitor re-enters the action on the next frame with the
// same candidates and the bot spends the wave starting and abandoning it.
stock void HoldOffAmmo(int actor)
{
	m_ctAmmoAsk[actor] = GetGameTime() + AMMO_GIVEUP_TIME;
	m_bAmmoPossible[actor] = false;
}

// IsPossible says whether there is ammo worth walking to.
stock bool CTFBotGetAmmo_IsPossible(int actor)
{
	// Skip lag.
	if ((m_iAmmoPack[actor] != -1) && IsValidAmmo(m_iAmmoPack[actor]))
	{
		return true;
	}
	if (m_ctAmmoAsk[actor] > GetGameTime())
	{
		return m_bAmmoPossible[actor];
	}
	m_ctAmmoAsk[actor] = GetGameTime() + AMMO_ASK_INTERVAL;
	ArrayList ammo = new ArrayList(2);
	ComputeHealthAndAmmoVectors(actor, ammo, tf_bot_ammo_search_range.FloatValue);
	bool bPossible = false;
	for (int i = 0; i < ammo.Length; i++)
	{
		if (!IsValidAmmo(ammo.Get(i, 0)))
		{
			continue;
		}
		bPossible = true;
		break;
	}
	m_bAmmoPossible[actor] = bPossible;
	delete ammo;
	return bPossible;
}

// ComputeVectors fills the list with what is in range, nearest first.
stock void ComputeHealthAndAmmoVectors(int client, ArrayList found, float maxRange)
{
	ArrayList nearby = new ArrayList(2);
	float myCentre[3];
	myCentre = WorldSpaceCenter(client);
	for (int i = 0; i < 5; i++)
	{
		int ammo = -1;
		for (;;)
		{
			ammo = FindEntityByClassname(ammo, g_strHealthAndAmmoEntities[i]);
			if (ammo == -1)
			{
				break;
			}
			// A wave leaves more of these on the floor than anybody is going to walk to
			if (nearby.Length >= HEALTH_CANDIDATES_MAX)
			{
				break;
			}
			if (BaseEntity_GetTeamNumber(ammo) == view_as<int>(GetPlayerEnemyTeam(client)))
			{
				continue;
			}
			float entityRange = GetVectorDistance(myCentre, WorldSpaceCenter(ammo));
			if (entityRange > maxRange)
			{
				continue;
			}
			if (BaseEntity_IsBaseObject(ammo))
			{
				// Can't get anything from still building buildings.
				if (TF2_IsBuilding(ammo))
				{
					continue;
				}
				// Skip empty dispenser.
				if ((TF2_GetObjectType(ammo) == TFObject_Dispenser) && (GetEntProp(ammo, Prop_Send, "m_iAmmoMetal") <= 0))
				{
					continue;
				}
			}
			int at = nearby.Push(entityRange);
			nearby.Set(at, ammo, 1);
		}
	}
	nearby.SortCustom(SortByStraightLineRange);
	int searches = 0;
	for (int i = 0; (i < nearby.Length) && (searches < HEALTH_PATHS_MAX); i++)
	{
		int ammo = nearby.Get(i, 1);
		searches++;
		float length;
		bool reachable = IsPathToVectorPossible(client, WorldSpaceCenter(ammo), length);
		if (!reachable)
		{
			continue;
		}
		if (length > maxRange)
		{
			continue;
		}
		int at = found.Push(ammo);
		found.Set(at, length, 1);
	}
	delete nearby;
}

// SortByStraightLineRange is the cheap ordering the search runs before it spends
// a nav mesh query on anything.
stock int SortByStraightLineRange(int index1, int index2, Handle array, Handle hndl)
{
	ArrayList list = view_as<ArrayList>(array);
	float first = view_as<float>(list.Get(index1, 0));
	float second = view_as<float>(list.Get(index2, 0));
	if (first < second)
	{
		return -1;
	}
	return (first > second ? 1 : 0);
}

// ResetGetAmmo forgets the ammo pack this bot was walking to.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetGetAmmo(int client)
{
	m_iAmmoPack[client] = -1;
	// The memo goes with it, and the hold-off above all
	//
	// IsPossible answers from a nav search around where the bot stood, and holds
	// the answer for half a second. HoldOff holds a no for three, which is what a
	// walk that ran out of reachable packs leaves behind. Neither is about the
	// seat: a bot that spawns into one is refused ammo it can reach, for a walk
	// its predecessor gave up on.
	m_ctAmmoAsk[client] = 0.0;
	m_bAmmoPossible[client] = false;
}

