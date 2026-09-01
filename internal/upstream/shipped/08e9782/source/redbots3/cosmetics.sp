/* Hats and unusual effects on the defender bots, for the look of the thing

None of this changes how a bot plays. What the team looks like now is six bare-headed mercenaries
with the same three weapons, and a run built out of a randomiser is more fun to watch when they
look like six strangers who met on the way in.

Both pools come from the game's own item schema through tf_econ_data, never from a table of
numbers in this file. A table of item definition indexes is a guess that goes stale on the next
update, and the schema is what the client renders from anyway. Each pool costs one walk of the
schema, once, and is then kept.

Attribute 134 attaches the particle, read out of scripts/items/items_game.txt rather than copied
from a forum post.

War paints on the weapons were here too and are gone: they are attributes on the entity the
upgrade station rewrites all wave, and they were not worth one more thing to blame a crash on.

Defender bots only, and never the invading robots: a wave is read by silhouette, and a robot in a
hat is a robot somebody shoots a moment later than they should. */

//The quality a client expects on an item before it will draw an effect on it
#define TF_QUALITY_UNIQUE		6
#define TF_QUALITY_UNUSUAL		5

#define ATTRIB_ATTACH_PARTICLE	134

//How many times a bot may draw again when what it drew has no model to wear
#define HAT_DRAW_TRIES			4

//What the schema calls the chest a tournament medal hangs on, and nothing worth looking at
#define EQUIP_REGION_MEDAL		"medal"

/* The schema's spelling of a class, which is not the mod's

items_game says heavy where the mod says heavyweapons, and the per class model of a hat is filed
under the schema's word. Index 0 is TFClass_Unknown. */
static const char ITEMS_GAME_CLASS[][] =
{
	"", "scout", "sniper", "soldier", "demoman", "medic", "heavy", "pyro", "spy", "engineer"
};

//One pool per class, because a hat one class can wear another cannot. Index 0 is TFClass_Unknown
static ArrayList g_adtHats[view_as<int>(TFClass_Engineer) + 1];
static ArrayList g_adtHatEffects;

/* What one bot wears, drawn once and worn for the rest of the mission

Drawn once and not per life, because the hat is how a player tells one bot from another. A team
of six that comes back from the respawn room in six new hats is a team nobody can follow, and
following them is most of what there is to do while they play.

The class is part of it: a bot that changes class between waves cannot wear what it drew, so it
draws again. */
enum struct Wardrobe
{
	bool drawn;
	TFClassType playerClass;
	int hatItem;
	int hatEffect;
}

static Wardrobe g_wardrobe[MAXPLAYERS + 1];

//The hat entity itself, which the game destroys on every respawn and this puts back
static int g_iBotHat[MAXPLAYERS + 1] = {INVALID_ENT_REFERENCE, ...};

//Whether a bot is already waiting to be dressed, so it is dressed once
static bool g_bCosmeticsPending[MAXPLAYERS + 1];

/* The item a bot is in the middle of putting on, and nothing the rest of the time

Equipping throws when the game refuses an item, and a thrown native takes the callback with it,
so there is no line after the call that can notice. What is left behind is this: an item still
written here on the next spawn is an item the game would not attach, and it comes out of the
pool rather than being drawn again for the rest of the map. */
static int g_iEquipping[MAXPLAYERS + 1];

/* Half a second after the bot spawns, not the moment it does

The game gives its own items on spawn, and the custom loadout replaces them a tenth of a second
later, and a hat handed to a bot in the middle of that is a hat the game throws away.

Once per spawn, however many times it is asked. A bot's first spawn asks twice: the spawn that
identifies it as ours, and the respawn that applies its loadout, which is close enough behind to
be the same moment. Without the flag that is a hat created, worn and taken off again for every
bot at the start of every wave. */
void GiveBotCosmeticsSoon(int client)
{
	if (!redbots_manager_bot_hats.BoolValue)
		return;
	
	if (g_bCosmeticsPending[client])
		return;
	
	g_bCosmeticsPending[client] = true;
	
	/* Spread across a second rather than all landing on the same frame
	
	A team spawns together, so six of these used to be scheduled for the same tenth of a second and
	fired on one frame. Dressing a bot creates an entity and precaches a model, and a precache
	that has to go to disk is not a thing to do six times inside one tick of a server that is also
	starting a wave. The half second is what it was; the rest is one bot after another. */
	float when = 0.5 + 0.15 * float(client % 8);
	
	CreateTimer(when, Timer_GiveBotCosmetics, GetClientUserId(client), TIMER_FLAG_NO_MAPCHANGE);
}

static Action Timer_GiveBotCosmetics(Handle timer, int userid)
{
	int client = GetClientOfUserId(userid);
	
	if (client < 1)
		return Plugin_Stop;
	
	g_bCosmeticsPending[client] = false;
	
	if (!IsClientInGame(client) || !IsPlayerAlive(client) || !g_bIsDefenderBot[client])
		return Plugin_Stop;
	
	/* Draw again when what the bot drew turns out not to be wearable, up to a few times

	A hat the schema has no model for is dropped from the pool and the bot was left bare until its
	next respawn, which for a defender that does not die is the rest of the mission. Every failure
	takes that item out of the pool, so the tries cannot chase the same bad one twice. */
	for (int attempt = 0; attempt < HAT_DRAW_TRIES; attempt++)
	{
		DrawWardrobe(client);
		
		if (!redbots_manager_bot_hats.BoolValue || WearHat(client))
			break;
	}
	
	return Plugin_Stop;
}

//What this bot is going to wear for the rest of the mission, drawn once
static void DrawWardrobe(int client)
{
	TFClassType playerClass = TF2_GetPlayerClass(client);
	
	if (g_iEquipping[client] != 0)
	{
		DropHatFromPool(g_wardrobe[client].playerClass, g_iEquipping[client]);
		g_iEquipping[client] = 0;
		g_wardrobe[client].drawn = false;
	}
	
	if (g_wardrobe[client].drawn && g_wardrobe[client].playerClass == playerClass)
		return;
	
	ArrayList hats = HatPoolForClass(playerClass);
	
	g_wardrobe[client].drawn = true;
	g_wardrobe[client].playerClass = playerClass;
	g_wardrobe[client].hatItem = hats != null && hats.Length > 0 ? hats.Get(GetRandomInt(0, hats.Length - 1)) : 0;
	g_wardrobe[client].hatEffect = redbots_manager_bot_hat_effects.BoolValue ? RandomHatEffect() : 0;
	
	if (redbots_manager_debug.BoolValue)
		LogMessage("[DrawWardrobe] %N drew item %d with effect %d", client, g_wardrobe[client].hatItem, g_wardrobe[client].hatEffect);
}

/* Put the drawn hat back on

The game clears a player's wearables every time it gives it its items, which is every respawn, so
the same hat has to be handed back rather than merely remembered. */
static bool WearHat(int client)
{
	RemoveBotHat(client);
	
	int itemDefinition = g_wardrobe[client].hatItem;
	int effect = g_wardrobe[client].hatEffect;
	
	if (itemDefinition < 1)
		return false;
	
	/* Cleared here rather than trusted to have been cleared

	TF2Util_EquipPlayerWearable throws on Peppy's server, "wearable entity 339 not attached to
	player", eighteen times in one session. The throw unwinds this function and the timer above it,
	so the line that clears g_iEquipping after the equip never runs and the next draw reads a refusal
	that belongs to the previous hat. Clearing on the way in is the one place the throw cannot skip.
	See the bead on the equip itself, which this does not fix. */
	g_iEquipping[client] = 0;

	int hat = CreateEntityByName("tf_wearable");
	
	if (hat == -1)
		return false;
	
	SetEntProp(hat, Prop_Send, "m_iItemDefinitionIndex", itemDefinition);
	SetEntProp(hat, Prop_Send, "m_bInitialized", 1);
	SetEntProp(hat, Prop_Send, "m_iEntityQuality", effect > 0 ? TF_QUALITY_UNUSUAL : TF_QUALITY_UNIQUE);
	SetEntProp(hat, Prop_Send, "m_iEntityLevel", 1);
	SetEntProp(hat, Prop_Send, "m_iTeamNum", GetClientTeam(client));
	
	/* The model, which the game does not work out for a wearable made by hand
	
	Without it the hat is an entity with no shape: the unusual effect drew, attached to nothing,
	and the bots wore particles and no hats. A hat with no model in the schema cannot be worn at
	all, so it goes the same way as one the game refuses. */
	char model[PLATFORM_MAX_PATH];
	
	if (!HatModel(itemDefinition, TF2_GetPlayerClass(client), model, sizeof(model)))
	{
		RemoveEntity(hat);
		DropHatFromPool(g_wardrobe[client].playerClass, itemDefinition);
		g_wardrobe[client].drawn = false;
		g_iBotHat[client] = INVALID_ENT_REFERENCE;
		
		return false;
	}
	
	SetEntityModel(hat, model);
	
	DispatchSpawn(hat);
	
	if (effect > 0)
		TF2Attrib_SetByDefIndex(hat, ATTRIB_ATTACH_PARTICLE, float(effect));
	
	//The game throws out a wearable whose item it thinks the wearer does not own, and a bot owns nothing
	TF2Util_SetWearableAlwaysValid(hat, true);
	
	/* Written down before the equip and not after, both of them
	
	The entity so that a refused one is taken away on the next spawn rather than standing in the
	world with nobody wearing it, and the item so that the refusal is noticed at all. */
	g_iBotHat[client] = EntIndexToEntRef(hat);
	g_iEquipping[client] = itemDefinition;
	
	TF2Util_EquipPlayerWearable(client, hat);
	
	g_iEquipping[client] = 0;
	
	if (redbots_manager_debug.BoolValue)
		LogMessage("[WearHat] %N puts item %d back on", client, itemDefinition);
	
	return true;
}

/* Wearables the game refused, standing in the world with nobody wearing them
 *
 * TF2Util_EquipPlayerWearable asserts that the wearable ended up attached, and throws when the
 * game declined it. A thrown native takes the rest of the callback with it, so the hat entity that
 * was created a few lines earlier is never cleaned up and never worn: it just stays. Seen in the
 * error log dozens of times across a mission, and a server that leaks an edict per bot per respawn
 * eventually runs out of them.
 *
 * The equip cannot be tested in advance, so the leak is swept instead of prevented. Anything of
 * ours whose owner is not a player in the game is nobody's hat.
 */
//As many as one pass will look at, and as many as it will take away
#define ORPHAN_WEARABLE_SWEEP	64

/* Collected first and removed afterwards, and the scan itself is what is bounded
 *
 * The first version of this removed as it walked and counted removals against the bound. Both
 * halves were wrong. FindEntityByClassname takes the previous entity as its cursor, and the
 * previous entity had just been deleted, so the walk restarted from the beginning; and a bound on
 * removals is not a bound on iterations, so a wearable somebody was wearing sent it round for
 * ever. The server's watchdog killed a run inside a minute.
 *
 * This file's own rule, from the top of the repository: every loop gets a fixed upper bound.
 */
void RemoveOrphanedWearables()
{
	int found[ORPHAN_WEARABLE_SWEEP];
	int count = 0;
	int seen = 0;
	
	int hat = -1;
	
	while ((hat = FindEntityByClassname(hat, "tf_wearable")) != -1 && ++seen <= ORPHAN_WEARABLE_SWEEP)
	{
		int owner = GetEntPropEnt(hat, Prop_Send, "m_hOwnerEntity");
		
		//Somebody is wearing it, or it is still being handed out this frame
		if (owner > 0)
			continue;
		
		if (count < ORPHAN_WEARABLE_SWEEP)
			found[count++] = EntIndexToEntRef(hat);
	}
	
	for (int i = 0; i < count; i++)
	{
		int orphan = EntRefToEntIndex(found[i]);
		
		if (orphan != INVALID_ENT_REFERENCE && IsValidEntity(orphan))
			RemoveEntity(orphan);
	}
	
	if (count > 0 && redbots_manager_debug.BoolValue)
		LogMessage("[cosmetics] swept %d wearable(s) nobody was wearing", count);
}

/* Take the hat off the way the game does it

Deleting the entity is not the same thing. The player holds a handle to every wearable it is
wearing, and an entity removed out from under that list leaves the game following a pointer to
something that is not there any more. */
static void RemoveBotHat(int client)
{
	int hat = EntRefToEntIndex(g_iBotHat[client]);
	
	g_iBotHat[client] = INVALID_ENT_REFERENCE;
	
	if (hat != INVALID_ENT_REFERENCE && IsClientInGame(client))
		TF2_RemoveWearable(client, hat);
}

/* The model this class wears this hat with

Per class first, because a hat that fits nine heads is nine models and the wrong one is a Scout
wearing a Heavy's hat at a Heavy's height. */
static bool HatModel(int itemDefinition, TFClassType playerClass, char[] model, int maxlength)
{
	/* Whether this class may wear it at all, asked before the model is looked up

	A hat with a generic model_player passes the lookup below for every class, and the game then
	refuses the equip: TF2Util_EquipPlayerWearable throws "wearable entity N not attached to player",
	eighteen times in one of Peppy's sessions. The throw unwinds Timer_GiveBotCosmetics, so the rest
	of that bot's cosmetics are lost as well.

	A loadout slot of -1 is the schema saying this class does not wear this item. Returning false
	here sends it down the same path as a hat with no model: dropped from the pool and drawn again,
	so the refusal costs a redraw rather than an exception. */
	if (TF2Econ_GetItemLoadoutSlot(itemDefinition, playerClass) == -1)
		return false;

	int index = view_as<int>(playerClass);
	
	if (index > 0 && index < sizeof(ITEMS_GAME_CLASS))
	{
		char key[64]; Format(key, sizeof(key), "model_player_per_class/%s", ITEMS_GAME_CLASS[index]);
		
		if (TF2Econ_GetItemDefinitionString(itemDefinition, key, model, maxlength) && model[0] != '\0')
			return PrecacheHatModel(model);
	}
	
	if (TF2Econ_GetItemDefinitionString(itemDefinition, "model_player", model, maxlength) && model[0] != '\0')
		return PrecacheHatModel(model);
	
	return false;
}

//A model the map did not load has to be added to the table before anything can wear it
static bool PrecacheHatModel(const char[] model)
{
	if (!IsModelPrecached(model))
		PrecacheModel(model);
	
	return true;
}

//An item the game will not attach, gone for the rest of the map so nobody draws it twice
static void DropHatFromPool(TFClassType playerClass, int itemDefinition)
{
	ArrayList pool = HatPoolForClass(playerClass);
	
	if (pool == null)
		return;
	
	int at = pool.FindValue(itemDefinition);
	
	if (at != -1)
		pool.Erase(at);
	
	LogMessage("Item %d cannot be worn by class %d and has been dropped from the hat pool", itemDefinition, playerClass);
}

/* A bot that has left takes its hat with it

Nothing to remove, then: the game clears a player's wearables when the player goes, and a
reference to an entity that is gone reads as no entity. Only what is remembered here is left. */
void ForgetBotCosmetics(int client)
{
	g_iBotHat[client] = INVALID_ENT_REFERENCE;
	g_bCosmeticsPending[client] = false;
	g_iEquipping[client] = 0;
	
	//The next bot in this seat is a different bot, and dresses itself
	g_wardrobe[client].drawn = false;
}

static int RandomHatEffect()
{
	if (g_adtHatEffects == null)
		g_adtHatEffects = TF2Econ_GetParticleAttributeList(ParticleSet_CosmeticUnusualEffects);
	
	if (g_adtHatEffects == null || g_adtHatEffects.Length < 1)
		return 0;
	
	return g_adtHatEffects.Get(GetRandomInt(0, g_adtHatEffects.Length - 1));
}

static ArrayList HatPoolForClass(TFClassType playerClass)
{
	int index = view_as<int>(playerClass);
	
	if (index < 1 || index >= sizeof(g_adtHats))
		return null;
	
	if (g_adtHats[index] == null)
		g_adtHats[index] = BuildHatPool(playerClass);
	
	return g_adtHats[index];
}

/* What every defender bot is actually wearing, entity by entity

A hat that does not show up is one of four things and they look the same from the outside: never
drawn, drawn and refused a model, worn by an entity the game threw away, or worn by an entity with
no model index on it. This says which. */
public Action Command_DumpHats(int client, int args)
{
	for (int playerClass = 1; playerClass < sizeof(g_adtHats); playerClass++)
	{
		ArrayList pool = HatPoolForClass(view_as<TFClassType>(playerClass));

		ReplyToCommand(client, "class %d: %d hats in the pool", playerClass, pool == null ? -1 : pool.Length);
	}

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !g_bIsDefenderBot[i])
			continue;

		int hat = EntRefToEntIndex(g_iBotHat[i]);

		if (hat == INVALID_ENT_REFERENCE)
		{
			ReplyToCommand(client, "%N: class %d, drew item %d, effect %d, wearing nothing",
				i, g_wardrobe[i].playerClass, g_wardrobe[i].hatItem, g_wardrobe[i].hatEffect);

			continue;
		}

		char model[PLATFORM_MAX_PATH]; GetEntPropString(hat, Prop_Data, "m_ModelName", model, sizeof(model));

		ReplyToCommand(client, "%N: class %d, item %d, effect %d, entity %d, owner %d, modelindex %d, model %s",
			i, g_wardrobe[i].playerClass, g_wardrobe[i].hatItem, g_wardrobe[i].hatEffect, hat,
			GetEntPropEnt(hat, Prop_Send, "m_hOwnerEntity"),
			GetEntProp(hat, Prop_Send, "m_nModelIndex"), model);
	}

	return Plugin_Handled;
}

/* Every cosmetic this class may wear, asked of the schema once

Not the medals. What the bots actually drew, almost every time, was a UGC participation medal or
an ozfortress season badge: a postage stamp on the chest that reads in game as a bot wearing
nothing at all, while anything with a particle on it still drew the particle, which is why the
effects looked like the only part that worked. There are far more tournament medals in the schema
than there are cosmetics, so drawing uniformly from the slot is drawing a medal.

Filed by equip region rather than by slot, because the slot cannot tell them apart: the head slot
is the game's old single-hat one and no modern item reports it, so every cosmetic and every medal
alike comes back as misc. The medals are the ones the schema puts in the "medal" region, which is
one string off a prefab they all share.

The class filter is the schema's own: an item a class cannot equip has no loadout slot for that
class. The classname filter keeps out the wearables that are really weapons, the demoman's shield
and the like, which have a cosmetic slot and are not worn on the body. */
static ArrayList BuildHatPool(TFClassType playerClass)
{
	ArrayList pool = new ArrayList();
	ArrayList items = TF2Econ_GetItemList();
	
	if (items == null)
		return pool;
	
	int miscSlot = TF2Econ_TranslateLoadoutSlotNameToIndex("misc");
	
	for (int i = 0; i < items.Length; i++)
	{
		int itemDefinition = items.Get(i);
		
		if (TF2Econ_GetItemLoadoutSlot(itemDefinition, playerClass) != miscSlot)
			continue;
		
		char equipRegion[64];
		
		if (TF2Econ_GetItemDefinitionString(itemDefinition, "equip_region", equipRegion, sizeof(equipRegion))
			&& StrEqual(equipRegion, EQUIP_REGION_MEDAL))
			continue;
		
		char className[64];
		
		if (!TF2Econ_GetItemClassName(itemDefinition, className, sizeof(className)))
			continue;
		
		if (!StrEqual(className, "tf_wearable"))
			continue;
		
		pool.Push(itemDefinition);
	}
	
	delete items;
	
	return pool;
}
