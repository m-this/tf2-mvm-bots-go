/* What a TF2 player says about itself.

Named send-table reads, plus the three condition sets the game treats as one
state: invulnerable, crit-boosted and stealthed are each several TFConds, and
asking for one of them means asking for all of its conditions. */

stock int TF2_GetCurrency(int client)
{
	return GetEntProp(client, Prop_Send, "m_nCurrency");
}

stock void TF2_SetCurrency(int client, int amount)
{
	SetEntProp(client, Prop_Send, "m_nCurrency", amount);
}

stock bool TF2_IsMiniBoss(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bIsMiniBoss"));
}

stock bool TF2_IsTaunting(int client)
{
	return TF2_IsPlayerInCondition(client, TFCond_Taunting);
}

stock bool TF2_IsInUpgradeZone(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bInUpgradeZone"));
}

stock void TF2_SetInUpgradeZone(int client, bool inZone)
{
	SetEntProp(client, Prop_Send, "m_bInUpgradeZone", inZone);
}

stock bool TF2_IsRageDraining(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bRageDraining"));
}

stock float TF2_GetRageMeter(int client)
{
	return GetEntPropFloat(client, Prop_Send, "m_flRageMeter");
}

stock bool TF2_IsCarryingObject(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bCarryingObject"));
}

stock int TF2_GetCarriedObject(int client)
{
	return GetEntPropEnt(client, Prop_Send, "m_hCarriedObject");
}

stock bool TF2_IsFeignDeathReady(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bFeignDeathReady"));
}

stock bool TF2_IsShieldEquipped(int client)
{
	return view_as<bool>(GetEntProp(client, Prop_Send, "m_bShieldEquipped"));
}

stock int TF2_GetNumHealers(int client)
{
	return GetEntProp(client, Prop_Send, "m_nNumHealers");
}

/* Carrying the bomb. m_hItem is the flag entity, and item_teamflag is the only
item TF2 hands a player this way. */
stock bool TF2_HasTheFlag(int client)
{
	int item = GetEntPropEnt(client, Prop_Send, "m_hItem");

	if (item == -1)
	{
		return false;
	}

	char classname[PLATFORM_MAX_PATH];
	GetEntityClassname(item, classname, sizeof(classname));

	return StrEqual(classname, "item_teamflag");
}

/* Ubered, by whichever of the four conditions did it. */
stock bool TF2_IsInvulnerable(int client)
{
	return TF2_IsPlayerInCondition(client, TFCond_Ubercharged)
		|| TF2_IsPlayerInCondition(client, TFCond_UberchargedCanteen)
		|| TF2_IsPlayerInCondition(client, TFCond_UberchargedHidden)
		|| TF2_IsPlayerInCondition(client, TFCond_UberchargedOnTakeDamage);
}

/* Critting with every weapon. The conditions that crit only one shot or only
one weapon are deliberately not here. */
stock bool TF2_IsCritBoosted(int client)
{
	return TF2_IsPlayerInCondition(client, TFCond_Kritzkrieged)
		|| TF2_IsPlayerInCondition(client, TFCond_HalloweenCritCandy)
		|| TF2_IsPlayerInCondition(client, TFCond_CritCanteen)
		|| TF2_IsPlayerInCondition(client, TFCond_CritOnFirstBlood)
		|| TF2_IsPlayerInCondition(client, TFCond_CritOnWin)
		|| TF2_IsPlayerInCondition(client, TFCond_CritOnFlagCapture)
		|| TF2_IsPlayerInCondition(client, TFCond_CritOnKill)
		|| TF2_IsPlayerInCondition(client, TFCond_CritOnDamage)
		|| TF2_IsPlayerInCondition(client, TFCond_CritRuneTemp);
}

stock bool TF2_IsStealthed(int client)
{
	return TF2_IsPlayerInCondition(client, TFCond_Cloaked)
		|| TF2_IsPlayerInCondition(client, TFCond_Stealthed)
		|| TF2_IsPlayerInCondition(client, TFCond_StealthedUserBuffFade);
}

stock TFTeam TF2_GetEnemyTeam(TFTeam team)
{
	if (team == TFTeam_Red)
	{
		return TFTeam_Blue;
	}

	if (team == TFTeam_Blue)
	{
		return TFTeam_Red;
	}

	return team;
}

/* The class names a popfile and a map configuration write, which are the class
index's own non-localised names.

The match is on the name's own length rather than the whole argument, because a
popfile writes "Heavyweapons" and "HeavyweaponsGiant" for the same class. */
static char s_szClassNames[][] =
{
	"Undefined",
	"Scout",
	"Sniper",
	"Soldier",
	"Demoman",
	"Medic",
	"Heavy",
	"Pyro",
	"Spy",
	"Engineer",
	"Civilian"
};

stock TFClassType TF2_GetClassIndexFromString(const char[] className)
{
	for (int i = 1; i < sizeof(s_szClassNames); i++)
	{
		int length = strlen(s_szClassNames[i]);

		if (strlen(className) < length)
		{
			continue;
		}

		if (strncmp(s_szClassNames[i], className, length, false) == 0)
		{
			return view_as<TFClassType>(i);
		}
	}

	return TFClass_Unknown;
}
