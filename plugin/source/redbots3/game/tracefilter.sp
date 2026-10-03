/* The rules a trace filter applies before it asks about line of fire.

Three questions the engine asks in CTraceFilterSimple and then in the game
rules: does the contents mask want this kind of entity at all, is the entity one
the trace is told to pass through, and do the two collision groups collide. The
tables are the Source and TF2 collision tables, read by group number. */

static stock int ModelTypeOf(int entity)
{
	char model[PLATFORM_MAX_PATH];
	GetEntPropString(entity, Prop_Data, "m_ModelName", model, sizeof(model));

	if (model[0] == '\0')
	{
		return mod_bad;
	}

	if (StrContains(model, ".spr", false) != -1 || StrContains(model, ".vmt", false) != -1)
	{
		return mod_sprite;
	}

	if (StrContains(model, ".bsp", false) != -1)
	{
		return mod_brush;
	}

	if (StrContains(model, ".mdl", false) != -1)
	{
		return mod_studio;
	}

	return mod_bad;
}

stock bool StandardFilterRules(int entity, int contentsMask)
{
	int solid = GetEntProp(entity, Prop_Data, "m_nSolidType");

	if (ModelTypeOf(entity) != mod_brush || (solid != SOLID_BSP && solid != SOLID_VPHYSICS))
	{
		if ((contentsMask & CONTENTS_MONSTER) == 0)
		{
			return false;
		}
	}

	if ((contentsMask & CONTENTS_WINDOW) == 0 && GetEntityRenderMode(entity) != RENDER_NORMAL)
	{
		return false;
	}

	if ((contentsMask & CONTENTS_MOVEABLE) == 0 && GetEntityMoveType(entity) == MOVETYPE_PUSH)
	{
		return false;
	}

	return true;
}

/* Neither the pass entity nor anything it owns, in either direction: a rocket
does not block its own shooter's line of fire. */
stock bool PassServerEntityFilter(int touch, int pass)
{
	if (touch == pass)
	{
		return false;
	}

	if (BaseEntity_GetOwnerEntity(touch) == pass)
	{
		return false;
	}

	if (BaseEntity_GetOwnerEntity(pass) == touch)
	{
		return false;
	}

	return true;
}

/* The collision table every Source game shares.

Both callers order the pair first, so the table only has to say what happens for
group0 <= group1. */
static stock bool SharedGroupsCollide(int group0, int group1)
{
	if ((group0 == COLLISION_GROUP_PLAYER || group0 == COLLISION_GROUP_PLAYER_MOVEMENT) && group1 == COLLISION_GROUP_PUSHAWAY)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_DEBRIS && group1 == COLLISION_GROUP_PUSHAWAY)
	{
		return true;
	}

	if (group0 == COLLISION_GROUP_IN_VEHICLE || group1 == COLLISION_GROUP_IN_VEHICLE)
	{
		return false;
	}

	if (group1 == COLLISION_GROUP_DOOR_BLOCKER && group0 != COLLISION_GROUP_NPC)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER && group1 == COLLISION_GROUP_PASSABLE_DOOR)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_DEBRIS || group0 == COLLISION_GROUP_DEBRIS_TRIGGER)
	{
		return false;
	}

	if ((group0 == COLLISION_GROUP_DISSOLVING || group1 == COLLISION_GROUP_DISSOLVING) && group0 != COLLISION_GROUP_NONE)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_INTERACTIVE_DEBRIS && group1 == COLLISION_GROUP_INTERACTIVE_DEBRIS)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_BREAKABLE_GLASS && group1 == COLLISION_GROUP_BREAKABLE_GLASS)
	{
		return false;
	}

	if (group1 == COLLISION_GROUP_INTERACTIVE && group0 != COLLISION_GROUP_NONE)
	{
		return false;
	}

	if (group1 == COLLISION_GROUP_PROJECTILE
		&& (group0 == COLLISION_GROUP_DEBRIS || group0 == COLLISION_GROUP_WEAPON || group0 == COLLISION_GROUP_PROJECTILE))
	{
		return false;
	}

	if (group1 == COLLISION_GROUP_WEAPON
		&& (group0 == COLLISION_GROUP_VEHICLE || group0 == COLLISION_GROUP_PLAYER || group0 == COLLISION_GROUP_NPC))
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_VEHICLE_CLIP || group1 == COLLISION_GROUP_VEHICLE_CLIP)
	{
		return group0 == COLLISION_GROUP_VEHICLE;
	}

	return true;
}

/* TF2's own collision table, which answers first and falls through to the shared
one for every pair it does not name. */
stock bool TFGameRules_ShouldCollide(int collisionGroup0, int collisionGroup1)
{
	int group0 = collisionGroup0 <= collisionGroup1 ? collisionGroup0 : collisionGroup1;
	int group1 = collisionGroup0 <= collisionGroup1 ? collisionGroup1 : collisionGroup0;

	bool rocket = group1 == TFCOLLISION_GROUP_ROCKETS
		|| group1 == TFCOLLISION_GROUP_ROCKET_BUT_NOT_WITH_OTHER_ROCKETS;

	if (group0 == COLLISION_GROUP_PLAYER_MOVEMENT && group1 == COLLISION_GROUP_WEAPON)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER_MOVEMENT && group1 == COLLISION_GROUP_PROJECTILE)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER && rocket)
	{
		return true;
	}

	if (group0 == COLLISION_GROUP_PLAYER_MOVEMENT && rocket)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_WEAPON && rocket)
	{
		return false;
	}

	if (group0 == TF_COLLISIONGROUP_GRENADES && rocket)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PROJECTILE && rocket)
	{
		return false;
	}

	if (group0 == TFCOLLISION_GROUP_ROCKETS && group1 == TFCOLLISION_GROUP_ROCKET_BUT_NOT_WITH_OTHER_ROCKETS)
	{
		return false;
	}

	if (group0 == TFCOLLISION_GROUP_ROCKET_BUT_NOT_WITH_OTHER_ROCKETS
		&& group1 == TFCOLLISION_GROUP_ROCKET_BUT_NOT_WITH_OTHER_ROCKETS)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER && group1 == TF_COLLISIONGROUP_GRENADES)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER_MOVEMENT && group1 == TF_COLLISIONGROUP_GRENADES)
	{
		return false;
	}

	if (group1 == TFCOLLISION_GROUP_RESPAWNROOMS)
	{
		return group0 == COLLISION_GROUP_PLAYER || group0 == COLLISION_GROUP_PLAYER_MOVEMENT;
	}

	if (group0 == TF_COLLISIONGROUP_GRENADES && group1 == TF_COLLISIONGROUP_GRENADES)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER_MOVEMENT && group1 == TFCOLLISION_GROUP_COMBATOBJECT)
	{
		return false;
	}

	if (group0 == COLLISION_GROUP_PLAYER && group1 == TFCOLLISION_GROUP_COMBATOBJECT)
	{
		return false;
	}

	if ((group0 == COLLISION_GROUP_PLAYER || group0 == COLLISION_GROUP_PLAYER_MOVEMENT) && group1 == TFCOLLISION_GROUP_TANK)
	{
		return false;
	}

	return SharedGroupsCollide(group0, group1);
}
