
#if !defined __tf_econ_data_included
#define TF_ITEMDEF_DEFAULT	-1
#endif


public Action Timer_GiveCustomLoadout(Handle timer, int client)
{
	if (!IsClientInGame(client))
		return Plugin_Stop;
	
	//These store weapon entity indexes so we can pass them later
	//PDA2 is excluded as it doesn't currently get upgrades
	int primary = -1, secondary = -1, melee = -1;
	
	char itemClassname[35];
	
	if (m_iWeaponPrimary[client] > TF_ITEMDEF_DEFAULT)
	{
		TF2_RemoveWeaponSlot(client, TFWeaponSlot_Primary);
		
		if (TF2Econ_GetItemClassName(m_iWeaponPrimary[client], itemClassname, sizeof(itemClassname)))
		{
			TF2Econ_TranslateWeaponEntForClass(itemClassname, sizeof(itemClassname), TF2_GetPlayerClass(client));
			primary = GiveItemToPlayer(client, itemClassname, m_iWeaponPrimary[client], 1, 6);
			
			if (g_bHasBoughtUpgrades[client] == false)
			{
				switch (m_iWeaponPrimary[client])
				{
					case 730:	TF2Attrib_SetByName(primary, "auto fires when full", 1.0); //Beggar's Bazooka: prevent overloading
					case 996:	TF2Attrib_SetByName(primary, "grenade launcher mortar mode", 0.0); //Loose Cannon: prevent charging
				}
			}
		}
		else
		{
			LogError("Timer_GiveCustomLoadout: Could not add primary %d to %N!", m_iWeaponPrimary[client], client);
		}
	}
	
	if (m_iWeaponSecondary[client] > TF_ITEMDEF_DEFAULT && !TF2_IsShieldEquipped(client))
	{
		TF2_RemoveWeaponSlot(client, TFWeaponSlot_Secondary);
		
		if (TF2Econ_GetItemClassName(m_iWeaponSecondary[client], itemClassname, sizeof(itemClassname)))
		{
			TF2Econ_TranslateWeaponEntForClass(itemClassname, sizeof(itemClassname), TF2_GetPlayerClass(client));
			secondary = GiveItemToPlayer(client, itemClassname, m_iWeaponSecondary[client], 1, 6);
			
			if (g_bHasBoughtUpgrades[client] == false && StrEqual(itemClassname, "tf_weapon_pipebomblauncher"))
				TF2Attrib_SetByName(secondary, "stickybomb charge rate", 0.0); //Instant fire stickies
		}
		else
		{
			LogError("Timer_GiveCustomLoadout: Could not add secondary %d to %N!", m_iWeaponSecondary[client], client);
		}
	}
	
	if (m_iWeaponMelee[client] > TF_ITEMDEF_DEFAULT)
	{
		TF2_RemoveWeaponSlot(client, TFWeaponSlot_Melee);
		
		if (TF2Econ_GetItemClassName(m_iWeaponMelee[client], itemClassname, sizeof(itemClassname)))
		{
			TF2Econ_TranslateWeaponEntForClass(itemClassname, sizeof(itemClassname), TF2_GetPlayerClass(client));
			melee = GiveItemToPlayer(client, itemClassname, m_iWeaponMelee[client], 1, 6);
			
			switch (m_iWeaponMelee[client])
			{
				case 1071:
				{
					//Dumb way to check, but we don't want to change these attributes every spawn
					if (g_bHasBoughtUpgrades[client] == false)
						GiveGoldPanStats(melee);
				}
			}
		}
		else
		{
			LogError("Timer_GiveCustomLoadout: Could not add melee %d to %N!", m_iWeaponMelee[client], client);
		}
	}
	
	if (m_iWeaponPDA2[client] > TF_ITEMDEF_DEFAULT)
	{
		TF2_RemoveWeaponSlot(client, TFWeaponSlot_Building);
		
		if (TF2Econ_GetItemClassName(m_iWeaponPDA2[client], itemClassname, sizeof(itemClassname)))
			GiveItemToPlayer(client, itemClassname, m_iWeaponPDA2[client], 1, 6);
		else
			LogError("Timer_GiveCustomLoadout: Could not add pda2 %d to %N!", m_iWeaponPDA2[client], client);
	}
	
	if (g_bHasBoughtUpgrades[client])
		ReapplyItemUpgrades(client, primary, secondary, melee);
	
	/* Certain weapons or upgrades may have changed our health and ammo
	so we must refill them completely to the max, though health
	may get reduced if max health was lowered by a weapon's attribute */
	for (int i = TF_AMMO_PRIMARY; i < TF_AMMO_COUNT; i++)
		GivePlayerAmmo(client, 1000, i, true);
	
	//For players, this is calculated max health
	int maxHealth = TF2Util_GetEntityMaxHealth(client);
	
	if (GetClientHealth(client) != maxHealth)
	{
		BaseEntity_SetMaxHealth(client, maxHealth);
		SetEntityHealth(client, maxHealth);
	}
	
	PostInventoryApplication(client);
	
	if (redbots_manager_debug.BoolValue)
		PrintToChatAll("[Timer_GiveCustomLoadout] %N's ammo: %d/%d", client, BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_PRIMARY), TF2Util_GetPlayerMaxAmmo(client, TF_AMMO_PRIMARY));
	
	return Plugin_Stop;
}

