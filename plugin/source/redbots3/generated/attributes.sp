/* Generated from internal/tables/attribute.go. Do not edit.

The table it comes from is the only place these names are written. */

/* The upgrade attribute names, and the id the ranking switches on

The ranking used to compare the attribute name against a chain of string literals, once per rank
it might return. The name is a text key from the item schema and the ranking is a pure function of
it, so the key becomes a number at the edge and the ranking becomes a switch.

An id is written down rather than counted, and never reused. A recorded run names the attribute it
ranked, so an id that changed meaning would silently re-read old results. */

enum
{
	ATTRIBUTE_NONE = 0,
	ATTRIBUTE_ATTACK_PROJECTILES = 1,
	ATTRIBUTE_GENERATE_RAGE_ON_HEAL = 2,
	ATTRIBUTE_WEAPON_BURN_DMG_INCREASED = 3,
	ATTRIBUTE_WEAPON_BURN_TIME_INCREASED = 4,
	ATTRIBUTE_MAXAMMO_METAL_INCREASED = 5,
	ATTRIBUTE_METAL_REGEN = 6,
	ATTRIBUTE_UBERCHARGE_RATE_BONUS = 7,
	ATTRIBUTE_HEALING_MASTERY = 8,
	ATTRIBUTE_DAMAGE_BONUS = 9,
	ATTRIBUTE_SRIFLE_CHARGE_RATE_INCREASED = 10,
	ATTRIBUTE_PROJECTILE_SPEED_INCREASED = 11,
	ATTRIBUTE_FIRE_RATE_BONUS = 12,
	ATTRIBUTE_CLIP_SIZE_UPGRADE_ATOMIC = 13,
	ATTRIBUTE_CLIP_SIZE_BONUS_UPGRADE = 14,
	ATTRIBUTE_FASTER_RELOAD_RATE = 15,
	ATTRIBUTE_MAXAMMO_PRIMARY_INCREASED = 16,
	ATTRIBUTE_MAXAMMO_SECONDARY_INCREASED = 17,
	ATTRIBUTE_ENGY_DISPENSER_RADIUS_INCREASED = 18,
	ATTRIBUTE_ENGY_SENTRY_FIRE_RATE_INCREASED = 19,
	ATTRIBUTE_ENGY_DISPOSABLE_SENTRIES = 20,
	ATTRIBUTE_ENGY_BUILDING_HEALTH_BONUS = 21,
	ATTRIBUTE_MELEE_ATTACK_RATE_BONUS = 22,
	ATTRIBUTE_UBER_DURATION_BONUS = 23,
	ATTRIBUTE_OVERHEAL_EXPERT = 24,
	ATTRIBUTE_EXPLOSIVE_SNIPER_SHOT = 25,
	ATTRIBUTE_ARMOR_PIERCING = 26,
	ATTRIBUTE_ROBO_SAPPER = 27,
	ATTRIBUTE_ROCKET_SPECIALIST = 28,
	ATTRIBUTE_HEAL_ON_KILL = 29,
	ATTRIBUTE_APPLIES_SNARE_EFFECT = 30,
	ATTRIBUTE_MAD_MILK_SYRINGES = 31,
	ATTRIBUTE_MOVE_SPEED_BONUS = 32,
	ATTRIBUTE_PROJECTILE_PENETRATION = 33,
	ATTRIBUTE_PROJECTILE_PENETRATION_HEAVY = 34,
	ATTRIBUTE_CRITBOOST_ON_KILL = 35,
	ATTRIBUTE_MARK_FOR_DEATH = 36,
	ATTRIBUTE_INCREASE_BUFF_DURATION = 37,
	ATTRIBUTE_EFFECT_BAR_RECHARGE_RATE_INCREASED = 38,
	ATTRIBUTE_CHARGE_RECHARGE_RATE_INCREASED = 39,
	ATTRIBUTE_GENERATE_RAGE_ON_DAMAGE = 40,
	ATTRIBUTE_BLEEDING_DURATION = 41,
	ATTRIBUTE_DMG_TAKEN_FROM_BLAST_REDUCED = 42,
	ATTRIBUTE_DMG_TAKEN_FROM_BULLETS_REDUCED = 43,
	ATTRIBUTE_DMG_TAKEN_FROM_FIRE_REDUCED = 44,
	ATTRIBUTE_HEALTH_REGEN = 45,
	ATTRIBUTE_DMG_TAKEN_FROM_CRIT_REDUCED = 46,
	ATTRIBUTE_DAMAGE_FORCE_REDUCTION = 47,
	ATTRIBUTE_INCREASED_JUMP_HEIGHT = 48,
};

#define ATTRIBUTE_COUNT 48

static const char g_strAttributeNames[ATTRIBUTE_COUNT][] =
{
	"attack projectiles",
	"generate rage on heal",
	"weapon burn dmg increased",
	"weapon burn time increased",
	"maxammo metal increased",
	"metal regen",
	"ubercharge rate bonus",
	"healing mastery",
	"damage bonus",
	"SRifle Charge rate increased",
	"Projectile speed increased",
	"fire rate bonus",
	"clip size upgrade atomic",
	"clip size bonus upgrade",
	"faster reload rate",
	"maxammo primary increased",
	"maxammo secondary increased",
	"engy dispenser radius increased",
	"engy sentry fire rate increased",
	"engy disposable sentries",
	"engy building health bonus",
	"melee attack rate bonus",
	"uber duration bonus",
	"overheal expert",
	"explosive sniper shot",
	"armor piercing",
	"robo sapper",
	"rocket specialist",
	"heal on kill",
	"applies snare effect",
	"mad milk syringes",
	"move speed bonus",
	"projectile penetration",
	"projectile penetration heavy",
	"critboost on kill",
	"mark for death",
	"increase buff duration",
	"effect bar recharge rate increased",
	"charge recharge rate increased",
	"generate rage on damage",
	"bleeding duration",
	"dmg taken from blast reduced",
	"dmg taken from bullets reduced",
	"dmg taken from fire reduced",
	"health regen",
	"dmg taken from crit reduced",
	"damage force reduction",
	"increased jump height",
};

/* The id for a schema attribute name, or ATTRIBUTE_NONE for one the ranking has no opinion about

ATTRIBUTE_NONE is a real answer and not a failure: the schema has hundreds of attributes and this
table holds the forty eight the ranking dispatches on. */
stock int AttributeID(const char[] name)
{
	for (int i = 0; i < ATTRIBUTE_COUNT; i++)
	{
		if (StrEqual(name, g_strAttributeNames[i]))
			return i + 1;
	}
	
	return ATTRIBUTE_NONE;
}
