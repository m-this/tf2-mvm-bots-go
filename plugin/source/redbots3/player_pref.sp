

/* Override all preferences to be from a specific player instead
This is meant to be changed by admins for personal use
The value here is the desired player's client index */
int g_iPlayerForcedPref = -1;

char g_sPlayerPrefPath[PLATFORM_MAX_PATH];
KeyValues m_kvPlayerPrefData;

/* A loadout the server sets for every bot, from configs/defenderbots/loadout.cfg
When the file exists it decides every slot: a slot it leaves out keeps the stock weapon,
and the players' own preferences are not consulted */
KeyValues m_kvServerLoadout;

/* The seat of sm_redbots_manager_team_composition a bot fills, counted from 1, and 0 for a bot that
fills none

The composition is an ordered list, so the third name in it is a seat somebody sits in and not just
another engineer. Looking a loadout up by class alone hands every engineer on the team the same two
weapons, which is the wrong answer the moment one of them is meant to hold the wrangler and the
other is not.

tf_bot_add is a console command, so the bot does not exist yet when its seat is decided. The seats
wait here in the order they were asked for and the next bot of ours to enter takes the one in front,
which is the order the server creates them in */

void Config_LoadServerLoadout()
{
	delete m_kvServerLoadout;

	//A map change, so the seats the last map asked for belong to bots that will never enter
	delete m_adtPendingBotSeats;

	char filePath[PLATFORM_MAX_PATH]; BuildPath(Path_SM, filePath, sizeof(filePath), "configs/defenderbots/loadout.cfg");

	if (!FileExists(filePath))
		return;

	m_kvServerLoadout = new KeyValues("loadout");

	if (!m_kvServerLoadout.ImportFromFile(filePath))
	{
		LogError("Config_LoadServerLoadout: Could not read %s!", filePath);
		delete m_kvServerLoadout;
		return;
	}

	WarnAboutInvalidLoadoutSeats();
}

/* Complain about the seats the file names that no bot can ever fill

/* Stand on the block the file writes for one seat, when it writes one this bot may wear

/* The seat decides the whole loadout when it names this bot, and the class decides it otherwise

static Action Timer_SavePrefData(Handle timer)
{
	if (!m_kvPlayerPrefData.ExportToFile(g_sPlayerPrefPath))
	{
		LogError("Timer_SavePrefData: Failed to save player preference data!");
		PrintToChatAll("%s ERROR: Player preference data failed to save!", PLUGIN_PREFIX);
		return Plugin_Continue;
	}
	
	if (redbots_manager_debug.BoolValue)
		PrintToServer("%s Saved player preference data.", PLUGIN_PREFIX);
	
	return Plugin_Continue;
}

void LoadPreferencesData()
{
	m_kvPlayerPrefData = new KeyValues("PlayerBotPreferences");
	m_kvPlayerPrefData.ImportFromFile(g_sPlayerPrefPath);
	
	CreateTimer(20.0, Timer_SavePrefData, _, TIMER_REPEAT);
}

/* void SetRandomWeaponPreference(int client, const char[] class, const char[] slot)
{
	if (StrEqual(class, "scout", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SCOUT_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_SCOUT_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SCOUT_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_SCOUT_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SCOUT_MELEE[GetRandomInt(0, sizeof(WEAPONS_SCOUT_MELEE) - 1)]);
	}
	else if (StrEqual(class, "soldier", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SOLDIER_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_SOLDIER_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SOLDIER_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_SOLDIER_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SOLDIER_MELEE[GetRandomInt(0, sizeof(WEAPONS_SOLDIER_MELEE) - 1)]);
	}
	else if (StrEqual(class, "pyro", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_PYRO_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_PYRO_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_PYRO_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_PYRO_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_PYRO_MELEE[GetRandomInt(0, sizeof(WEAPONS_PYRO_MELEE) - 1)]);
	}
	else if (StrEqual(class, "demoman", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_DEMOMAN_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_DEMOMAN_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_DEMOMAN_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_DEMOMAN_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_DEMOMAN_MELEE[GetRandomInt(0, sizeof(WEAPONS_DEMOMAN_MELEE) - 1)]);
	}
	else if (StrEqual(class, "heavyweapons", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_HEAVY_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_HEAVY_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_HEAVY_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_HEAVY_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_HEAVY_MELEE[GetRandomInt(0, sizeof(WEAPONS_HEAVY_MELEE) - 1)]);
	}
	else if (StrEqual(class, "engineer", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_ENGINEER_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_ENGINEER_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_ENGINEER_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_ENGINEER_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_ENGINEER_MELEE[GetRandomInt(0, sizeof(WEAPONS_ENGINEER_MELEE) - 1)]);
	}
	else if (StrEqual(class, "medic", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_MEDIC_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_MEDIC_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_MEDIC_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_MEDIC_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_MEDIC_MELEE[GetRandomInt(0, sizeof(WEAPONS_MEDIC_MELEE) - 1)]);
	}
	else if (StrEqual(class, "sniper", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SNIPER_PRIMARY[GetRandomInt(0, sizeof(WEAPONS_SNIPER_PRIMARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SNIPER_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_SNIPER_SECONDARY) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SNIPER_MELEE[GetRandomInt(0, sizeof(WEAPONS_SNIPER_MELEE) - 1)]);
	}
	else if (StrEqual(class, "spy", false))
	{
		if (StrEqual(slot, "primary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SPY_SECONDARY[GetRandomInt(0, sizeof(WEAPONS_SPY_SECONDARY) - 1)]);
		else if (StrEqual(slot, "secondary", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SPY_BUILDING[GetRandomInt(0, sizeof(WEAPONS_SPY_BUILDING) - 1)]);
		else if (StrEqual(slot, "melee", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SPY_MELEE[GetRandomInt(0, sizeof(WEAPONS_SPY_MELEE) - 1)]);
		else if (StrEqual(slot, "pda2", false))
			SetWeaponPreference(client, class, slot, WEAPONS_SPY_PDA2[GetRandomInt(0, sizeof(WEAPONS_SPY_PDA2) - 1)]);
	}
	else
	{
		PrintToChatAll("[SetRandomWeaponPreference] Unknown class of %s", class);
		LogError("SetRandomWeaponPreference: Unknown class %s", class);
	}
} */

void ShowCurrentBotClassChances(int client = -1)
{
	const int maxClassCount = view_as<int>(TFClass_Engineer);
	
	//Each index is for a class, 0 = scout, 1 = soldier, etc.
	//Defined as float for percentage calculation later down below
	float classChoiceCount[maxClassCount];
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && IsValidForBotPreferences(i))
		{
			int classFlags = GetClassPreferencesFlags(i);
			
			if (classFlags & PREF_FL_SCOUT)
				classChoiceCount[0]++;
			
			if (classFlags & PREF_FL_SOLDIER)
				classChoiceCount[1]++;
			
			if (classFlags & PREF_FL_PYRO)
				classChoiceCount[2]++;
			
			if (classFlags & PREF_FL_DEMO)
				classChoiceCount[3]++;
			
			if (classFlags & PREF_FL_HEAVY)
				classChoiceCount[4]++;
			
			if (classFlags & PREF_FL_ENGINEER)
				classChoiceCount[5]++;
			
			if (classFlags & PREF_FL_MEDIC)
				classChoiceCount[6]++;
			
			if (classFlags & PREF_FL_SNIPER)
				classChoiceCount[7]++;
			
			if (classFlags & PREF_FL_SPY)
				classChoiceCount[8]++;
		}
	}
	
	float totalChoices;
	
	for (int i = 0; i < sizeof(classChoiceCount); i++)
		totalChoices += classChoiceCount[i];
	
	if (totalChoices == 0.0)
	{
		if (client > 0)
			PrintHintText(client, "Nobody has any preferences!");
		else
			PrintHintTextToAll("Nobody has any preferences!");
		
		return;
	}
	
	//Like before, each index represents a class
	float classPercents[maxClassCount];
	
	//Class percentage is amount of the class chosen divided by total of all class choices
	for (int i = 0; i < sizeof(classPercents); i++)
		classPercents[i] = (classChoiceCount[i] / totalChoices) * 100;
	
	if (client > 0)
	{
		CreateDisplayPanelBotPercentages(client, classPercents);
	}
	else
	{
		for (int i = 1; i <= MaxClients; i++)
			if (IsClientInGame(i))
				CreateDisplayPanelBotPercentages(i, classPercents);
	}
}