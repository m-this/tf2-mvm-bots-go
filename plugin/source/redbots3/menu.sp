

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
