/* What an entity says about itself, through its send table and its datamap.

Every function here is one named property read or written. The names are the
game's: m_iHealth, m_hOwnerEntity, m_iTeamNum. Nothing in this file decides
anything, so a change in behaviour can only come from a property name being
wrong, which spcomp and the server say out loud. */

stock bool BaseEntity_IsPlayer(int entity)
{
	return entity > 0 && entity <= MaxClients;
}

stock int BaseEntity_GetTeamNumber(int entity)
{
	return GetEntProp(entity, Prop_Send, "m_iTeamNum");
}

stock int BaseEntity_GetHealth(int entity)
{
	return GetEntProp(entity, Prop_Data, "m_iHealth");
}

stock void BaseEntity_SetMaxHealth(int entity, int amount)
{
	SetEntProp(entity, Prop_Data, "m_iMaxHealth", amount);
}

stock int BaseEntity_GetOwnerEntity(int entity)
{
	return GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
}

stock int BaseEntity_GetCollisionGroup(int entity)
{
	return GetEntProp(entity, Prop_Send, "m_CollisionGroup");
}

stock void BaseEntity_GetAbsOrigin(int entity, float buffer[3])
{
	GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", buffer);
}

stock void BaseEntity_GetLocalOrigin(int entity, float buffer[3])
{
	GetEntPropVector(entity, Prop_Data, "m_vecOrigin", buffer);
}

/* An eye position for anything, which for a player is the one the engine keeps
and for everything else is the origin plus the view offset. */
stock void BaseEntity_EyePosition(int entity, float buffer[3])
{
	if (BaseEntity_IsPlayer(entity))
	{
		GetClientEyePosition(entity, buffer);
		return;
	}

	float origin[3]; GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
	float offset[3]; GetEntPropVector(entity, Prop_Data, "m_vecViewOffset", offset);

	AddVectors(origin, offset, buffer);
}

/* The middle of the collision box, which is where to aim at something that is
not a player and so has no eyes to aim at. */
stock void BaseEntity_WorldSpaceCenter(int entity, float buffer[3])
{
	float origin[3]; GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
	float mins[3]; GetEntPropVector(entity, Prop_Data, "m_vecMins", mins);
	float maxs[3]; GetEntPropVector(entity, Prop_Data, "m_vecMaxs", maxs);

	float offset[3]; AddVectors(mins, maxs, offset);
	ScaleVector(offset, 0.5);
	AddVectors(origin, offset, buffer);
}

/* A building, told apart by a field only CBaseObject's datamap carries. There is
no class name to test against: every building has its own. */
stock bool BaseEntity_IsBaseObject(int entity)
{
	return HasEntProp(entity, Prop_Data, "CBaseObjectUpgradeThink");
}

/* The two entities TF2 counts as combat items, which a trace filter drops when
they belong to the shooter's own team. */
stock bool BaseEntity_IsCombatItem(int entity)
{
	char classname[PLATFORM_MAX_PATH];
	GetEdictClassname(entity, classname, sizeof(classname));

	return StrEqual(classname, "entity_medigun_shield") || StrEqual(classname, "entity_revive_marker");
}

stock void BaseEntity_MarkNeedsNamePurge(int entity)
{
	SetEntProp(entity, Prop_Data, "m_bForcePurgeFixedupStrings", true);
}

stock float BaseAnimating_GetModelScale(int entity)
{
	return GetEntPropFloat(entity, Prop_Data, "m_flModelScale");
}

stock int BaseCombatCharacter_GetActiveWeapon(int entity)
{
	return GetEntPropEnt(entity, Prop_Send, "m_hActiveWeapon");
}

stock int BaseCombatCharacter_GetAmmoCount(int entity, int ammoIndex)
{
	if (ammoIndex == -1)
	{
		return 0;
	}

	return GetEntProp(entity, Prop_Data, "m_iAmmo", _, ammoIndex);
}

stock void BaseCombatCharacter_RemoveAmmo(int entity, int count, int ammoIndex)
{
	if (count <= 0)
	{
		return;
	}

	int left = GetEntProp(entity, Prop_Data, "m_iAmmo", _, ammoIndex) - count;
	SetEntProp(entity, Prop_Data, "m_iAmmo", left > 0 ? left : 0, _, ammoIndex);
}

stock int BasePlayer_GetObserverMode(int client)
{
	return GetEntProp(client, Prop_Data, "m_iObserverMode");
}

stock void BasePlayer_EyeVectors(int client, float vecForward[3], float vecRight[3] = NULL_VECTOR, float vecUp[3] = NULL_VECTOR)
{
	float angles[3]; GetClientEyeAngles(client, angles);
	GetAngleVectors(angles, vecForward, vecRight, vecUp);
}
