/*
Package cosmetics is source/redbots3/cosmetics.sp: the hats and the unusual
effects on the defender bots.

None of it changes how a bot plays. The two callbacks stay behind in the plugin,
because a timer and a console command are function references and the subset has
none: what is here is everything they call.
*/
package cosmetics

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// Classes is one more than the last class, so the array is indexed by class.
const Classes = 10

// QualityUnique is the quality an item carries when nothing is attached to it.
//
//sp:name TF_QUALITY_UNIQUE
const QualityUnique = 6

// QualityUnusual is the quality a client expects before it will draw an effect.
//
//sp:name TF_QUALITY_UNUSUAL
const QualityUnusual = 5

// AttribAttachParticle is attribute 134, which attaches the particle.
//
//sp:name ATTRIB_ATTACH_PARTICLE
const AttribAttachParticle = 134

/*
OrphanWearableSweep is as many wearables as one pass looks at, and as many as it
takes away.

The scan is what is bounded rather than the removals: a bound on removals is not
a bound on iterations, and a wearable somebody was wearing sent the first version
round for ever.
*/
//
//sp:name ORPHAN_WEARABLE_SWEEP
const OrphanWearableSweep = 64

// EquipRegionMedal is what the schema calls the chest a tournament medal hangs
// on, and nothing worth looking at.
//
//sp:name EQUIP_REGION_MEDAL
const EquipRegionMedal = "medal"

/*
Wardrobe is what one bot wears, drawn once and worn for the rest of the mission.

Drawn once and not per life, because the hat is how a player tells one bot from
another. The class is part of it: a bot that changes class between waves cannot
wear what it drew, so it draws again.
*/
type Wardrobe struct {
	Drawn       bool
	PlayerClass engine.Class
	HatItem     int32
	HatEffect   int32
}

/*
The schema's spelling of a class, which is not the mod's.

items_game says heavy where the mod says heavyweapons, and the per class model of
a hat is filed under the schema's word. Index 0 is TFClass_Unknown.
*/
//
//sp:name ITEMS_GAME_CLASS
var itemsGameClass = [Classes]string{
	"", "scout", "sniper", "soldier", "demoman", "medic", "heavy", "pyro", "spy", "engineer",
}

// One pool per class, because a hat one class can wear another cannot.
//
//sp:name g_adtHats
var hats [Classes]engine.List

//sp:name g_adtHatEffects
var hatEffects engine.List

//sp:name g_wardrobe
var wardrobe [Slots]Wardrobe

// The hat entity itself, which the game destroys on every respawn and this puts
// back.
//
//sp:name g_iBotHat
var botHat = [Slots]int32{engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference()}

// Whether a bot is already waiting to be dressed, so it is dressed once.
//
//sp:name g_bCosmeticsPending
var cosmeticsPending [Slots]bool

/*
The item a bot is in the middle of putting on, and nothing the rest of the time.

Equipping throws when the game refuses an item, and a thrown native takes the
callback with it, so there is no line after the call that can notice. An item
still written here on the next spawn is one the game would not attach, and it
comes out of the pool.
*/
//
//sp:name g_iEquipping
var equipping [Slots]int32

/*
DrawWardrobe is what this bot is going to wear for the rest of the mission.
*/
//
//sp:name DrawWardrobe
func DrawWardrobe(client int32) {
	playerClass := engine.PlayerClass(client)

	/* An item still written here is one the game refused, not one with no model

	Both reasons end in DropHatFromPool and it logs the same line for each, so
	the log could not tell a hat the schema has no model for from one the game
	would not attach. Only the second throws, and only the second is mvm-6gi.
	Named here because this is the one place that knows which it was. */
	if equipping[client] != 0 {
		engine.LogMessage("Item %d passed the schema and the game refused to attach it to class %d, which is mvm-6gi", equipping[client], wardrobe[client].PlayerClass)
		DropHatFromPool(wardrobe[client].PlayerClass, equipping[client])
		equipping[client] = 0
		wardrobe[client].Drawn = false
	}

	if wardrobe[client].Drawn && wardrobe[client].PlayerClass == playerClass {
		return
	}

	pool := HatPoolForClass(playerClass)

	wardrobe[client].Drawn = true
	wardrobe[client].PlayerClass = playerClass

	if pool != engine.NoList() && pool.Length() > 0 {
		wardrobe[client].HatItem = pool.Get(engine.RandomInt(0, pool.Length()-1))
	} else {
		wardrobe[client].HatItem = 0
	}

	if engine.BotHatEffects().Bool() {
		wardrobe[client].HatEffect = RandomHatEffect()
	} else {
		wardrobe[client].HatEffect = 0
	}

	if engine.ManagerDebug().Bool() {
		engine.LogMessage("[DrawWardrobe] %N drew item %d with effect %d", client, wardrobe[client].HatItem, wardrobe[client].HatEffect)
	}
}

/*
WearHat puts the drawn hat back on.

The game clears a player's wearables every time it gives it its items, which is
every respawn, so the same hat has to be handed back rather than merely
remembered.
*/
//
//sp:name WearHat
func WearHat(client int32) bool {
	RemoveBotHat(client)

	itemDefinition := wardrobe[client].HatItem
	effect := wardrobe[client].HatEffect

	if itemDefinition < 1 {
		return false
	}

	/* Cleared here rather than trusted to have been cleared

	TF2Util_EquipPlayerWearable throws when the game refuses the item, and the
	throw unwinds this function and the timer above it, so the line that clears
	this after the equip never runs. Clearing on the way in is the one place the
	throw cannot skip. */
	equipping[client] = 0

	// The quality decides whether a client draws the effect at all.
	quality := int32(QualityUnique)

	if effect > 0 {
		quality = QualityUnusual
	}

	hat := engine.CreateEntityByName("tf_wearable")

	if hat == -1 {
		return false
	}

	engine.SetEntPropSend(hat, engine.PropSend(), "m_iItemDefinitionIndex", itemDefinition)
	engine.SetEntPropSend(hat, engine.PropSend(), "m_bInitialized", 1)

	engine.SetEntPropSend(hat, engine.PropSend(), "m_iEntityQuality", quality)

	engine.SetEntPropSend(hat, engine.PropSend(), "m_iEntityLevel", 1)
	engine.SetEntPropSend(hat, engine.PropSend(), "m_iTeamNum", engine.GetClientTeam(client))

	/* The model, which the game does not work out for a wearable made by hand

	Without it the hat is an entity with no shape: the unusual effect drew,
	attached to nothing. A hat with no model in the schema cannot be worn at all,
	so it goes the same way as one the game refuses. */
	var model engine.Text

	if !HatModel(itemDefinition, engine.PlayerClass(client), model, 512) {
		engine.RemoveEntity(hat)
		DropHatFromPool(wardrobe[client].PlayerClass, itemDefinition)
		wardrobe[client].Drawn = false
		botHat[client] = engine.InvalidEntReference()

		return false
	}

	engine.SetEntityModel(hat, model)

	engine.DispatchSpawn(hat)

	if effect > 0 {
		engine.SetAttributeByDefIndex(hat, AttribAttachParticle, float32(effect))
	}

	// The game throws out a wearable whose item it thinks the wearer does not
	// own, and a bot owns nothing.
	engine.SetWearableAlwaysValid(hat, true)

	/* Written down before the equip and not after, both of them

	The entity so that a refused one is taken away on the next spawn rather than
	standing in the world with nobody wearing it, and the item so that the
	refusal is noticed at all. */
	botHat[client] = engine.EntIndexToEntRef(hat)
	equipping[client] = itemDefinition

	engine.EquipPlayerWearable(client, hat)

	equipping[client] = 0

	if engine.ManagerDebug().Bool() {
		engine.LogMessage("[WearHat] %N puts item %d back on", client, itemDefinition)
	}

	return true
}

/*
RemoveOrphanedWearables sweeps up the wearables the game refused, which stand in
the world with nobody wearing them.

The equip cannot be tested in advance, so the leak is swept rather than
prevented. Anything of ours whose owner is not a player in the game is nobody's
hat. Collected first and removed afterwards: FindEntityByClassname takes the
previous entity as its cursor, and a cursor that has just been deleted restarts
the walk from the beginning.
*/
//
//sp:name RemoveOrphanedWearables
func RemoveOrphanedWearables() {
	var found [OrphanWearableSweep]int32

	count := int32(0)
	seen := int32(0)

	hat := int32(-1)

	for {
		hat = engine.FindEntityByClassname(hat, "tf_wearable")
		seen++

		if hat == -1 || seen > OrphanWearableSweep {
			break
		}

		owner := engine.EntPropEnt(hat, engine.PropSend(), "m_hOwnerEntity")

		// Somebody is wearing it, or it is still being handed out this frame.
		if owner > 0 {
			continue
		}

		if count < OrphanWearableSweep {
			found[count] = engine.EntIndexToEntRef(hat)
			count++
		}
	}

	for i := int32(0); i < count; i++ {
		orphan := engine.EntRefToEntIndex(found[i])

		if orphan != engine.InvalidEntReference() && engine.IsValidEntity(orphan) {
			engine.RemoveEntity(orphan)
		}
	}

	if count > 0 && engine.ManagerDebug().Bool() {
		engine.LogMessage("[cosmetics] swept %d wearable(s) nobody was wearing", count)
	}
}

/*
RemoveBotHat takes the hat off the way the game does it.

Deleting the entity is not the same thing: the player holds a handle to every
wearable it is wearing, and an entity removed out from under that list leaves the
game following a pointer to something that is not there any more.
*/
//
//sp:name RemoveBotHat
func RemoveBotHat(client int32) {
	hat := engine.EntRefToEntIndex(botHat[client])

	botHat[client] = engine.InvalidEntReference()

	if hat != engine.InvalidEntReference() && engine.IsClientInGame(client) {
		engine.RemoveWearable(client, hat)
	}
}

/*
HatModel is the model this class wears this hat with.

Per class first, because a hat that fits nine heads is nine models and the wrong
one is a Scout wearing a Heavy's hat at a Heavy's height.

A loadout slot of -1 is the schema saying this class does not wear this item, and
it is asked before the model is looked up: a hat with a generic model_player
passes the lookup for every class and the game then refuses the equip, which
throws and takes the rest of that bot's cosmetics with it.
*/
//
//sp:name HatModel
//sp:length model maxlength
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func HatModel(itemDefinition int32, playerClass engine.Class, model engine.Text, maxlength int32) bool {
	if engine.ItemLoadoutSlot(itemDefinition, playerClass) == -1 {
		return false
	}

	index := int32(playerClass)

	if index > 0 && index < Classes {
		key := engine.Format("model_player_per_class/%s", itemsGameClass[index])

		if engine.ItemDefinitionStringInto(itemDefinition, key, model, maxlength) && model[0] != 0 {
			return PrecacheHatModel(model)
		}
	}

	if engine.ItemDefinitionStringKeyInto(itemDefinition, "model_player", model, maxlength) && model[0] != 0 {
		return PrecacheHatModel(model)
	}

	return false
}

// PrecacheHatModel adds a model the map did not load to the table, which
// anything has to be in before it can be worn.
//
//sp:name PrecacheHatModel
func PrecacheHatModel(model engine.Text) bool {
	if !engine.IsModelPrecached(model) {
		engine.PrecacheModel(model)
	}

	return true
}

// DropHatFromPool takes an item the game will not attach out for the rest of the
// map, so nobody draws it twice.
//
//sp:name DropHatFromPool
func DropHatFromPool(playerClass engine.Class, itemDefinition int32) {
	pool := HatPoolForClass(playerClass)

	if pool == engine.NoList() {
		return
	}

	at := pool.FindValue(itemDefinition)

	if at != -1 {
		pool.Erase(at)
	}

	engine.LogMessage("Item %d cannot be worn by class %d and has been dropped from the hat pool", itemDefinition, playerClass)
}

/*
ForgetBotCosmetics drops what is remembered about a bot that has left.

Nothing to remove: the game clears a player's wearables when the player goes, and
a reference to an entity that is gone reads as no entity.
*/
//
//sp:name ForgetBotCosmetics
func ForgetBotCosmetics(client int32) {
	botHat[client] = engine.InvalidEntReference()
	cosmeticsPending[client] = false
	equipping[client] = 0

	// The next bot in this seat is a different bot, and dresses itself.
	wardrobe[client].Drawn = false
}

// RandomHatEffect is one of the game's own unusual effects.
//
//sp:name RandomHatEffect
func RandomHatEffect() int32 {
	if hatEffects == engine.NoList() {
		hatEffects = engine.ParticleAttributeList(engine.CosmeticUnusualEffects())
	}

	if hatEffects == engine.NoList() || hatEffects.Length() < 1 {
		return 0
	}

	return hatEffects.Get(engine.RandomInt(0, hatEffects.Length()-1))
}

// HatPoolForClass is what that class may wear, built the first time it is asked.
//
//sp:name HatPoolForClass
//sp:borrowed
func HatPoolForClass(playerClass engine.Class) engine.List {
	index := int32(playerClass)

	if index < 1 || index >= Classes {
		return engine.NoList()
	}

	if hats[index] == engine.NoList() {
		hats[index] = BuildHatPool(playerClass)
	}

	return hats[index]
}

/*
BuildHatPool is every cosmetic this class may wear, asked of the schema once.

Not the medals. What the bots drew almost every time was a tournament medal: a
postage stamp on the chest that reads in game as a bot wearing nothing at all.
There are far more medals in the schema than cosmetics, so drawing uniformly from
the slot is drawing a medal.

Filed by equip region rather than by slot, because the slot cannot tell them
apart: no modern item reports the old head slot, so cosmetics and medals alike
come back as misc. The class filter is the schema's own, and the classname filter
keeps out the wearables that are really weapons.
*/
//
//sp:name BuildHatPool
func BuildHatPool(playerClass engine.Class) (pool engine.List) {
	pool = engine.NewList()

	items := engine.ItemList()

	if items == engine.NoList() {
		return pool
	}

	miscSlot := engine.LoadoutSlotNameToIndex("misc")

	for i := int32(0); i < items.Length(); i++ {
		itemDefinition := items.Get(i)

		if engine.ItemLoadoutSlot(itemDefinition, playerClass) != miscSlot {
			continue
		}

		hasRegion, equipRegion := engine.ItemDefinitionString(itemDefinition, "equip_region")

		if hasRegion && engine.StrEqual(equipRegion, EquipRegionMedal) {
			continue
		}

		named, className := engine.ItemClassName(itemDefinition)

		if !named {
			continue
		}

		if !engine.StrEqual(className, "tf_wearable") {
			continue
		}

		pool.Push(itemDefinition)
	}

	items.Close()

	return pool
}

// HatDrawTries is how many times a bot may draw again when what it drew has no
// model to wear.
//
//sp:name HAT_DRAW_TRIES
const HatDrawTries = 4

/*
GiveBotCosmeticsSoon dresses the bot half a second after it spawns, not the
moment it does.

The game gives its own items on spawn and the custom loadout replaces them a
tenth of a second later, and a hat handed to a bot in the middle of that is a hat
the game throws away. Once per spawn, however many times it is asked: a bot's
first spawn asks twice.

Spread across a second rather than all landing on the same frame. A team spawns
together, and dressing a bot creates an entity and precaches a model, which is
not a thing to do six times inside one tick of a server that is also starting a
wave.
*/
//
//sp:name GiveBotCosmeticsSoon
func GiveBotCosmeticsSoon(client int32) {
	if !engine.BotHats().Bool() {
		return
	}

	if cosmeticsPending[client] {
		return
	}

	cosmeticsPending[client] = true

	when := 0.5 + 0.15*float32(client%8)

	engine.CreateTimerWith(when, TimerGiveBotCosmetics, engine.ClientUserID(client), engine.TimerNoMapChange())
}

/*
TimerGiveBotCosmetics is the dressing itself, once the spawn has settled.

Draw again when what the bot drew turns out not to be wearable, up to a few
times. A hat the schema has no model for is dropped from the pool and the bot was
left bare until its next respawn, which for a defender that does not die is the
rest of the mission. Every failure takes that item out of the pool, so the tries
cannot chase the same bad one twice.
*/
//
//sp:name Timer_GiveBotCosmetics
//sp:public
//
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerGiveBotCosmetics(timer engine.Timer, userid int32) engine.Outcome {
	client := engine.ClientOfUserID(userid)

	if client < 1 {
		return engine.PluginStop()
	}

	cosmeticsPending[client] = false

	if !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) || !engine.DefenderBotFlag(client) {
		return engine.PluginStop()
	}

	for attempt := int32(0); attempt < HatDrawTries; attempt++ {
		DrawWardrobe(client)

		if !engine.BotHats().Bool() || WearHat(client) {
			break
		}
	}

	return engine.PluginStop()
}

/*
CommandDumpHats says what every defender bot is actually wearing, entity by
entity.

A hat that does not show up is one of four things and they look the same from
outside: never drawn, drawn and refused a model, worn by an entity the game
threw away, or worn by an entity with no model index on it. This says which.
*/
//
//sp:name Command_DumpHats
//sp:public
//
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandDumpHats(client int32, args int32) engine.Outcome {
	for playerClass := int32(1); playerClass < Classes; playerClass++ {
		pool := HatPoolForClass(engine.Class(playerClass))

		engine.ReplyToCommand(client, "class %d: %d hats in the pool", playerClass, engine.ChooseInt(pool == engine.NoList(), -1, pool.Length()))
	}

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.DefenderBotFlag(i) {
			continue
		}

		hat := engine.EntRefToEntIndex(botHat[i])

		if hat == engine.InvalidEntReference() {
			engine.ReplyToCommand(client, "%N: class %d, drew item %d, effect %d, wearing nothing",
				i, wardrobe[i].PlayerClass, wardrobe[i].HatItem, wardrobe[i].HatEffect)

			continue
		}

		model := engine.EntPropString(hat, engine.PropData(), "m_ModelName")

		engine.ReplyToCommand(client, "%N: class %d, item %d, effect %d, entity %d, owner %d, modelindex %d, model %s",
			i, wardrobe[i].PlayerClass, wardrobe[i].HatItem, wardrobe[i].HatEffect, hat,
			engine.EntPropEnt(hat, engine.PropSend(), "m_hOwnerEntity"),
			engine.EntProp(hat, engine.PropSend(), "m_nModelIndex"), model)
	}

	return engine.PluginHandled()
}
