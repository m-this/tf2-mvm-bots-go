/* What a building says about itself.

All send-table reads but the last two: detonating one is the RemoveHealth input
with the building's own health, which is how the game reaches CBaseObject::Killed
rather than deleting the entity, and setting a teleporter's mode has to write the
key value as well as the property because the model and the sound follow it. */

stock bool TF2_IsBuilding(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bBuilding"));
}

stock bool TF2_IsPlacing(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bPlacing"));
}

stock bool TF2_IsCarried(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bCarried"));
}

stock bool TF2_HasSapper(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bHasSapper"));
}

stock bool TF2_IsPlasmaDisabled(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bPlasmaDisable"));
}

stock bool TF2_IsMiniBuilding(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bMiniBuilding"));
}

stock bool TF2_IsDisposableBuilding(int entity)
{
	return view_as<bool>(GetEntProp(entity, Prop_Send, "m_bDisposableBuilding"));
}

stock int TF2_GetUpgradeLevel(int entity)
{
	return GetEntProp(entity, Prop_Send, "m_iUpgradeLevel");
}

stock int TF2_GetMaxUpgradeLevel()
{
	return OBJ_MAX_UPGRADE_LEVEL;
}

stock void TF2_SetObjectMode(int entity, TFObjectMode mode)
{
	if (TF2_GetObjectType(entity) == TFObject_Teleporter)
	{
		DispatchKeyValue(entity, "teleporterType", mode == TFObjectMode_Entrance ? "1" : "2");
	}

	SetEntProp(entity, Prop_Send, "m_iObjectMode", mode);
}

stock void TF2_DetonateObject(int entity)
{
	SetVariantInt(BaseEntity_GetHealth(entity));
	AcceptEntityInput(entity, "RemoveHealth", entity, entity);
}
