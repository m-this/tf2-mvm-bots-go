void InitGameEventHooks()
{
	HookEvent("player_spawn", Event_PlayerSpawn);
	HookEvent("mvm_wave_failed", Event_MvmWaveFailed);
	HookEvent("mvm_wave_complete", Event_MvmWaveComplete);
	HookEvent("revive_player_notify", Event_RevivePlayerNotify);
	HookEvent("mvm_begin_wave", Event_MvmWaveBegin);
	HookEvent("player_team", Event_PlayerTeam);
	HookEvent("mvm_mission_update", Event_MvmMissionUpdate, EventHookMode_Pre);
	HookEvent("teamplay_round_start", Event_TeamplayRoundStart);
	HookEvent("player_death", Event_PlayerDeath);
}

/* The medic call, which the game gives as a voice command and not as an event

There is no player_calls_for_medic. Hooking that name is what this tried first, and HookEvent
throws on an event the game does not define, which took OnPluginStart down with it: the mod failed
to load, RED filled with nobody, and the run refused itself. A listener cannot fail that way.

"voicemenu 0 0" is MEDIC!. The first menu is the one with the call in it and the first entry is the
call. Passed straight through: this reads the command and does not answer it. */
Action Listener_VoiceMenu(int client, const char[] command, int argc)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
		return Plugin_Continue;

	if (argc < 2 || IsTFBotPlayer(client))
		return Plugin_Continue;

	char menu[4]; GetCmdArg(1, menu, sizeof(menu));
	char entry[4]; GetCmdArg(2, entry, sizeof(entry));

	if (StringToInt(menu) == 0 && StringToInt(entry) == 0)
		NoteMedicCall(client);

	return Plugin_Continue;
}

static void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	
	if (TF2_GetClientTeam(client) == TFTeam_Red && IsTFBotPlayer(client))
		CreateTimer(0.2, Timer_PlayerSpawn, client, TIMER_FLAG_NO_MAPCHANGE);
	
	//The popfile is still building this robot, so the bend waits a frame
	if (TF2_GetClientTeam(client) == TFTeam_Blue && IsFakeClient(client))
		BluAssist_OnRobotSpawn(client);
	
	if (g_bIsDefenderBot[client])
	{
		GiveBotCosmeticsSoon(client);
		
		g_bIsBeingRevived[client] = false;
		g_iBuyUpgradesNumber[client] = CanBuyUpgradesNow(client) ? GetRandomInt(1, 100) : 0;
		
		if (redbots_manager_debug.BoolValue)
			PrintToChatAll("[Event_PlayerSpawn] g_iBuyUpgradesNumber[%d] = %d", client, g_iBuyUpgradesNumber[client]);
	}
}

static void Event_MvmWaveComplete(Event event, const char[] name, bool dontBroadcast)
{
	OpenTheBreak();

	//A lineup retyped mid-wave was held until now
	Reseat_OnBreak();

	/* Before anything below sends the engineers off to shop
	The upgrade session is what tears their buildings down, and it needs this answer to know whether
	it should */
	EngineerNestRelocation_OnWaveComplete();
	
	if (redbots_manager_kick_bots.BoolValue)
	{
		RemoveAllDefenderBots("BotManager3: Wave complete!", IsFinalWave());
		ManageDefenderBots(false);
		CreateTimer(0.1, Timer_UpdateChosenBotTeamComposition, _, TIMER_FLAG_NO_MAPCHANGE);
		PrintToChatAll("%s Use command !viewbotlineup to view the next bot team composition", PLUGIN_PREFIX);
	}
	
#if defined MOD_REQUEST_CREDITS
	bool bRequestCredits = redbots_manager_bot_request_credits.BoolValue;
#endif
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && g_bIsDefenderBot[i])
		{
			//Wave complete, rethink what we should do
			ClearSniperStall(i);
			ResetIntentionInterface(i);
			
#if defined MOD_REQUEST_CREDITS
			if (bRequestCredits)
				FakeClientCommand(i, "sm_requestcredits");
#endif
		}
	}
}

/* Every defender rethinks what it is doing when a wave begins, one of them per tick

Resetting the intention throws away a bot's behaviour and has it rebuilt on its next update, and
rebuilding runs the OnStart of whatever it picks. Several of those are not cheap: MoveToFront
walks every prop_dynamic on the map and computes a path to each robot hologram, GetHealth and
GetAmmo search for something to walk to, the engineer scores a nest.

Doing that for six bots inside the wave_begin frame puts all of it on the one frame of a mission
that is already the most expensive: every robot spawns there and starts pathing at the same
moment, which is what "NextBot tickrate changed from 0 to 7" in the console is. Three runs of an
A/B died on exactly that frame, and the watchdog does not care that the work is finite.

So the resets are a queue and the queue is drained a bot a tick. The wave is minutes long and the
queue is at most the server's player count, which is a rounding error against it. The same shape,
and the same reason, as the nest relocation evaluator. */

static void Event_PlayerTeam(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	TFTeam team = view_as<TFTeam>(event.GetInt("team"));
	TFTeam oldTeam = view_as<TFTeam>(event.GetInt("oldteam"));
	bool isDisconnect = event.GetBool("disconnect");
	
	if (!IsFakeClient(client))
	{
		/* When changing teams, update bot team composition for
		- red player disconnected
		- player joined red
		- player left red */
		if ((isDisconnect && oldTeam == TFTeam_Red) || (!isDisconnect && (team == TFTeam_Red || oldTeam == TFTeam_Red)))
		{
			CreateTimer(0.1, Timer_UpdateChosenBotTeamComposition, _, TIMER_FLAG_NO_MAPCHANGE);
			
			if (oldTeam == TFTeam_Red)
			{
				HandleTeamPlayerCountChanged(TFTeam_Red, client);
			}
		}
		
#if defined CHANGETEAM_RESTRICTIONS
		if (!isDisconnect && team == TFTeam_Red && oldTeam == TFTeam_Blue && !CheckCommandAccess(client, NULL_STRING, ADMFLAG_GENERIC, true))
		{
			//Switching from BLUE to RED will temporarily ban the player from starting the bots
			if (g_flEnableBotsCooldown[client] <= GetGameTime())
				g_flEnableBotsCooldown[client] = GetGameTime() + 30.0;
			else
				g_flEnableBotsCooldown[client] += 10.0;
		}
#endif
	}
}

/* A Spy who kills somebody has told the team he exists

The cheapest honest sighting there is, and the one that matters: a team that has just lost
somebody to a knife knows where the knife was. Everything the bots do about Spies grows out of
this and out of seeing one undisguised, and nothing else */
static void Event_PlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	int victim = GetClientOfUserId(event.GetInt("userid"));
	
	if (!IsValidClientIndex(attacker) || !IsValidClientIndex(victim) || attacker == victim)
		return;
	
	if (TF2_GetPlayerClass(attacker) != TFClass_Spy)
		return;

	/* A robot's Spy, and not the team's own
	
	The rule was "a Spy killed somebody on another team", which a defending Spy satisfies every time
	he stabs a robot. So the team's own Spy put the whole team on alert, and the bots then frisked
	each other and him. Reported from play by somebody trying to play Spy: "your teammates keep
	trying to call you out as an enemy spy".
	
	Measured before the fix: a lineup with a friendly Spy in it spent 5.4 per cent of its samples
	spy checking, and two lineups without one spent none at all across eight thousand samples. */
	if (TF2_GetClientTeam(attacker) != TFTeam_Blue)
		return;
	
	float origin[3]; GetClientAbsOrigin(victim, origin);
	
	NoteSpySighting(origin);
}

static Action Timer_PlayerSpawn(Handle timer, int data)
{
	if (!IsClientInGame(data) || !IsTFBotPlayer(data) || TF2_GetClientTeam(data) != TFTeam_Red)
		return Plugin_Stop;
	
	if (g_bIsDefenderBot[data])
	{
#if defined MOD_REQUEST_CREDITS
		//Mainly for wave failures, try to request credits again
		if (redbots_manager_bot_request_credits.BoolValue && GameRules_GetRoundState() == RoundState_BetweenRounds)
			FakeClientCommand(data, "sm_requestcredits");
#endif
		
		if (redbots_manager_debug.BoolValue)
			PrintToChatAll("[Timer_PlayerSpawn] %N's currency: %d", data, TF2_GetCurrency(data));
		
		//We already made this guy into our bot, so do nothing
		return Plugin_Stop;
	}
	
	char clientName[MAX_NAME_LENGTH]; GetClientName(data, clientName, sizeof(clientName));
	
	//Identify if the bot is ours
	if (StrContains(clientName, TFBOT_IDENTITY_NAME) != -1)
	{
		g_bIsDefenderBot[data] = true;
		g_bHasBoughtUpgrades[data] = false;
		
		//The spawn that identified this bot ran before the flag above was set, so its cosmetics were skipped
		GiveBotCosmeticsSoon(data);
		
		if (redbots_manager_use_custom_loadouts.BoolValue)
		{
			//NOTE: for some reason, custom weapons aren't given unless the player respawns again
			TF2_RespawnPlayer(data);
		}
		else
		{
			//Not using custom loadouts, so we will only ever be using a sniper rifle
			//NOTE: custom loadouts runs it own check for the sniper's primary
			if (TF2_GetPlayerClass(data) == TFClass_Sniper)
				SetMission(data, CTFBot_MISSION_SNIPER);
		}
		
		//Let medic bots use their shields
		VS_AddBotAttribute(data, CTFBot_PROJECTILE_SHIELD);
		
		BaseEntity_MarkNeedsNamePurge(data);
		
		//Bots don't get their credits set when joining red because CTFGameRules::GetTeamAssignmentOverride ignores bot players
		//Set their credits manually to what they should have like human players
		//
		//The third term is Archipelago's. The first two come from the game's own record of the
		//wave, which never saw a Cash Bundle: without it a bot that rejoins or changes class
		//comes back with every bundle it was paid missing. Zero on a server without the plugin.
		SetCurrencyWithBundles(data, GetStartingCurrency(g_iPopulationManager) + GetAcquiredCreditsOfAllWaves());
		
		//Set the bot's field-of-view to 90
		//Its vision FOV will update in CTFBotMainAction::Update based on the property m_iFOV
		SetFakeClientConVar(data, "fov_desired", "90");
		
		SDKHook(data, SDKHook_TouchPost, DefenderBot_TouchPost);
		
		DHooks_DefenderBot(data);
		
#if defined IDLEBOT_AIMING
		//In this build we handle the bot's aiming manually, so don't have any of its nextbot aiming interfere with ours
		VS_AddBotAttribute(data, CTFBot_IGNORE_ENEMIES);
#endif
		
#if defined MOD_REQUEST_CREDITS
		if (redbots_manager_bot_request_credits.BoolValue)
			FakeClientCommand(data, "sm_requestcredits");
#endif
		
#if defined MOD_CUSTOM_ATTRIBUTES
		if (TF2Attrib_IsValidAttributeName("cannot be sapped"))
			TF2Attrib_SetByName(data, "cannot be sapped", 1.0);
#endif
		
		SetRandomNameOnBot(data);
	}
	
	return Plugin_Stop;
}

