

bool StartBotVote(int voteCaller)
{
	Menu vMenu = CreateMenu(MenuHandler_BotVote, MENU_ACTIONS_ALL);
	SetMenuTitle(vMenu, "%N wants to enable bots for this round.\nBots will fill in for missing teammates.", voteCaller);
	AddMenuItem(vMenu, "0", "Add bots for this game.");
	AddMenuItem(vMenu, "1", "Don't add bots this game.");
	SetMenuExitButton(vMenu, false);
	
	PrintToChatAll("%s A player started a bot game vote.", PLUGIN_PREFIX);
	
	int total = 0;
	int[] players = new int[MaxClients];
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && TF2_GetClientTeam(i) == TFTeam_Red)
			players[total++] = i;
	
	if (VoteMenu(vMenu, players, total, 15))
	{
		//Remember who started the vote
		g_iUIDBotSummoner = GetClientUserId(voteCaller);
		return true;
	}
	
	return false;
}
void CreateDisplayPanelBotPercentages(int client, float classPercents[TFClass_Engineer], const int duration = 30)
{
	if (IsFakeClient(client))
		return;
	
	Panel hPanel = new Panel();
	hPanel.SetTitle("Defender Bot Class Chances");
	
	char itemText[128];
	
	if (classPercents[0] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Scout: %.0f%%", classPercents[0]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[1] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Soldier: %.0f%%", classPercents[1]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[2] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Pyro: %.0f%%", classPercents[2]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[3] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Demoman: %.0f%%", classPercents[3]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[4] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Heavy: %.0f%%", classPercents[4]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[5] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Engineer: %.0f%%", classPercents[5]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[6] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Medic: %.0f%%", classPercents[6]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[7] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Sniper: %.0f%%", classPercents[7]);
		hPanel.DrawItem(itemText);
	}
	
	if (classPercents[8] > 0.0)
	{
		Format(itemText, sizeof(itemText), "Spy: %.0f%%", classPercents[8]);
		hPanel.DrawItem(itemText);
	}
	
	hPanel.Send(client, MenuHandler_ShowBotChances, duration);
	
	delete hPanel;
}

bool CreateDisplayPanelBotTeamComposition(int client, const int duration = 30)
{
	if (g_adtChosenBotClasses.Length == 0)
		return false;
	
	Panel hPanel = new Panel();
	hPanel.SetTitle("Current Bot Lineup");
	
	char itemText[TF2_CLASS_MAX_NAME_LENGTH];
	
	for (int i = 0; i < g_adtChosenBotClasses.Length; i++)
	{
		g_adtChosenBotClasses.GetString(i, itemText, sizeof(itemText));
		hPanel.DrawItem(itemText);
	}
	
	bool bSuccess = hPanel.Send(client, MenuHandler_ShowBotTeamComposition, duration);
	
	delete hPanel;
	
	return bSuccess;
}
