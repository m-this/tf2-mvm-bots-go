/* The item an entity is, and the weapon it holds.

m_iItemID has no send-table entry of its own: it is the 64-bit field the two
halves CEconEntity does publish are cut out of, so it is reached by the offset of
the high half less four bytes. Writing a fresh identifier is what makes the game
treat a handed-out weapon as its own item rather than a copy of somebody's. */

static int s_offsetItemID = -1;

stock void EconItemView_SetItemID(int entity, int id)
{
	if (s_offsetItemID < 0)
	{
		s_offsetItemID = FindSendPropInfo("CEconEntity", "m_iItemIDHigh") - 4;
	}

	SetEntData(entity, s_offsetItemID, id);

	SetEntProp(entity, Prop_Send, "m_iItemIDHigh", id >> 32);
	SetEntProp(entity, Prop_Send, "m_iItemIDLow", id & 0xFFFFFFFF);
}

stock bool WeaponID_IsSniperRifle(int weaponID)
{
	return weaponID == TF_WEAPON_SNIPERRIFLE
		|| weaponID == TF_WEAPON_SNIPERRIFLE_DECAP
		|| weaponID == TF_WEAPON_SNIPERRIFLE_CLASSIC;
}
