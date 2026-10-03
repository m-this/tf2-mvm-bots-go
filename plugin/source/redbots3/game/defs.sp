/* Constants the game itself numbers.

Netprop enumerations, indices into m_iAmmo, collision groups: values TF2 and the
Source engine fix, not decisions this plugin makes. They are written out here so
the plugin reads them from its own tree rather than from a third-party include.

The names are the engine's and the SDK's, because a server operator reading a
netprop dump beside this code has to be able to match the two up.

Untagged on purpose. A tagged enum would make every call site that passes one of
these to an int parameter warn, and nothing here is a type. */

#define COMMAND_MAX_RATE			0.3
#define MVM_BUYBACK_COST_PER_SEC	5
#define TF_FLAGINFO_HOME			0
#define MAX_CONTROL_POINTS			8
#define MAX_PREVIOUS_POINTS			3
#define OBJ_MAX_UPGRADE_LEVEL		3
#define TF_GAMETYPE_ARENA			4

/* The largest finite float, as its bit pattern. There is no literal for it. */
#define FLT_MAX						view_as<float>(0x7F7FFFFF)

/* CTFBot's bot-attribute flags, of which the plugin sets one. */
#define CTFBot_PROJECTILE_SHIELD	(1 << 27)

/* m_iObserverMode. */
enum
{
	OBS_MODE_NONE = 0,
	OBS_MODE_DEATHCAM,
	OBS_MODE_FREEZECAM,
	OBS_MODE_FIXED,
	OBS_MODE_IN_EYE,
	OBS_MODE_CHASE,
	OBS_MODE_POI,
	OBS_MODE_ROAMING,
	NUM_OBSERVER_MODES
};

/* m_nSolidType, through CCollisionProperty::GetSolid. */
enum
{
	SOLID_NONE = 0,
	SOLID_BSP,
	SOLID_BBOX,
	SOLID_OBB,
	SOLID_OBB_YAW,
	SOLID_CUSTOM,
	SOLID_VPHYSICS,
	SOLID_LAST
};

/* What a model is, which here is decided by the model path's extension. */
enum
{
	mod_bad = 0,
	mod_brush,
	mod_sprite,
	mod_studio
};

/* m_CollisionGroup: the groups every Source game shares. */
enum
{
	COLLISION_GROUP_NONE = 0,
	COLLISION_GROUP_DEBRIS,
	COLLISION_GROUP_DEBRIS_TRIGGER,
	COLLISION_GROUP_INTERACTIVE_DEBRIS,
	COLLISION_GROUP_INTERACTIVE,
	COLLISION_GROUP_PLAYER,
	COLLISION_GROUP_BREAKABLE_GLASS,
	COLLISION_GROUP_VEHICLE,
	COLLISION_GROUP_PLAYER_MOVEMENT,
	COLLISION_GROUP_NPC,
	COLLISION_GROUP_IN_VEHICLE,
	COLLISION_GROUP_WEAPON,
	COLLISION_GROUP_VEHICLE_CLIP,
	COLLISION_GROUP_PROJECTILE,
	COLLISION_GROUP_DOOR_BLOCKER,
	COLLISION_GROUP_PASSABLE_DOOR,
	COLLISION_GROUP_DISSOLVING,
	COLLISION_GROUP_PUSHAWAY,
	COLLISION_GROUP_NPC_ACTOR,
	COLLISION_GROUP_NPC_SCRIPTED,
	LAST_SHARED_COLLISION_GROUP
};

/* And the ones TF2 adds, which continue the shared numbering. */
enum
{
	TF_COLLISIONGROUP_GRENADES = LAST_SHARED_COLLISION_GROUP,
	TFCOLLISION_GROUP_OBJECT,
	TFCOLLISION_GROUP_OBJECT_SOLIDTOPLAYERMOVEMENT,
	TFCOLLISION_GROUP_COMBATOBJECT,
	TFCOLLISION_GROUP_ROCKETS,
	TFCOLLISION_GROUP_RESPAWNROOMS,
	TFCOLLISION_GROUP_TANK,
	TFCOLLISION_GROUP_ROCKET_BUT_NOT_WITH_OTHER_ROCKETS
};

/* Indices into m_iAmmo. */
enum
{
	TF_AMMO_DUMMY = 0,
	TF_AMMO_PRIMARY,
	TF_AMMO_SECONDARY,
	TF_AMMO_METAL,
	TF_AMMO_GRENADES1,
	TF_AMMO_GRENADES2,
	TF_AMMO_GRENADES3,
	TF_AMMO_COUNT
};

/* The power-up bottle's charge types, as m_usingPowerupBottle numbers them. */
enum
{
	POWERUP_BOTTLE_NONE = 0,
	POWERUP_BOTTLE_CRITBOOST,
	POWERUP_BOTTLE_UBERCHARGE,
	POWERUP_BOTTLE_RECALL,
	POWERUP_BOTTLE_REFILL_AMMO,
	POWERUP_BOTTLE_BUILDINGS_INSTANT_UPGRADE,
	POWERUP_BOTTLE_RADIUS_STEALTH,
	POWERUP_BOTTLE_TOTAL
};

/* Which part of the upgrade screen an mvm_upgrades.txt entry belongs to. */
enum
{
	UIGROUP_UPGRADE_ATTACHED_TO_ITEM = 0,
	UIGROUP_UPGRADE_ATTACHED_TO_PLAYER,
	UIGROUP_POWERUPBOTTLE
};

/* m_nMission on a CTFBot. */
enum
{
	CTFBot_NO_MISSION = 0,
	CTFBot_MISSION_SEEK_AND_DESTROY,
	CTFBot_MISSION_DESTROY_SENTRIES,
	CTFBot_MISSION_SNIPER,
	CTFBot_MISSION_SPY,
	CTFBot_MISSION_ENGINEER,
	CTFBot_MISSION_REPROGRAMMED
};
