package engine

/*
The item schema and the wearables made out of it.

Everything a hat needs: the schema the pool is built from, the entity the hat is,
and the two natives that attach it to a player. tf_econ_data answers the schema
questions and there is no other source for them, which is why none of this is a
table of item definition indexes.
*/

// CosmeticCalls are the answers.
type CosmeticCalls struct {
	ItemList                    func() List
	ItemLoadoutSlot             func(itemDefinition int32, playerClass Class) int32
	ItemDefinitionString        func(itemDefinition int32, key string) (bool, Text)
	ItemDefinitionStringAt      func(itemDefinition int32, key Text) (bool, Text)
	ItemDefinitionStringInto    func(itemDefinition int32, key Text, out Text, maxlength int32) bool
	ItemDefinitionStringKeyInto func(itemDefinition int32, key string, out Text, maxlength int32) bool
	ItemClassName               func(itemDefinition int32) (bool, Text)
	LoadoutSlotNameToIndex      func(name string) int32
	ParticleAttributeList       func(set ParticleSet) List
	SetAttributeByDefIndex      func(entity int32, attribute int32, value float32)
	SetWearableAlwaysValid      func(wearable int32, valid bool)
	RemoveWearable              func(client int32, wearable int32)
	SetEntityModel              func(entity int32, model Text)
	RemoveEntity                func(entity int32)
	IsModelPrecached            func(model Text) bool
	PrecacheModel               func(model Text) int32
	ListErase                   func(l List, index int32)
	ListFindValue               func(l List, value int32) int32
	BotHats                     func() ConVar
	BotHatEffects               func() ConVar
}

var cosmetics CosmeticCalls

// InstallCosmetics puts a set of answers behind them.
func InstallCosmetics(c CosmeticCalls) func() {
	previous := cosmetics
	Fill(&c)
	cosmetics = c
	return func() { cosmetics = previous }
}

// ParticleSet is tf_econ_data's TFParticleSet.
//
//sp:tag TFParticleSet
type ParticleSet int32

// CosmeticUnusualEffects is the set of unusual effects a hat may carry.
//
//sp:global ParticleSet_CosmeticUnusualEffects
func CosmeticUnusualEffects() ParticleSet { return 0 }

// ItemList is every item definition in the schema. The caller owns it.
//
//sp:native TF2Econ_GetItemList
func ItemList() List { return cosmetics.ItemList() }

// ItemLoadoutSlot is the slot that class wears that item in, or -1 when it
// cannot wear it at all.
//
//sp:native TF2Econ_GetItemLoadoutSlot
func ItemLoadoutSlot(itemDefinition int32, playerClass Class) int32 {
	return cosmetics.ItemLoadoutSlot(itemDefinition, playerClass)
}

// ItemDefinitionString reads one key off the item's schema entry.
//
//sp:native TF2Econ_GetItemDefinitionString sized
func ItemDefinitionString(itemDefinition int32, key string) (found bool, out Text) {
	return cosmetics.ItemDefinitionString(itemDefinition, key)
}

// ItemDefinitionStringInto reads a key into a buffer the caller owns, which is
// how a function that was handed one passes it on rather than copying.
//
//sp:native TF2Econ_GetItemDefinitionString
func ItemDefinitionStringInto(itemDefinition int32, key Text, out Text, maxlength int32) bool {
	return cosmetics.ItemDefinitionStringInto(itemDefinition, key, out, maxlength)
}

// ItemDefinitionStringKeyInto is the same with the key written out.
//
//sp:native TF2Econ_GetItemDefinitionString
func ItemDefinitionStringKeyInto(itemDefinition int32, key string, out Text, maxlength int32) bool {
	return cosmetics.ItemDefinitionStringKeyInto(itemDefinition, key, out, maxlength)
}

// ItemDefinitionStringAt is the same native where the key was built rather than
// written out, which is how the per class model is looked up.
//
//sp:native TF2Econ_GetItemDefinitionString sized
func ItemDefinitionStringAt(itemDefinition int32, key Text) (found bool, out Text) {
	return cosmetics.ItemDefinitionStringAt(itemDefinition, key)
}

// ItemClassName is the entity classname the item spawns as.
//
//sp:native TF2Econ_GetItemClassName sized
func ItemClassName(itemDefinition int32) (found bool, out Text) {
	return cosmetics.ItemClassName(itemDefinition)
}

// LoadoutSlotNameToIndex turns the schema's word for a slot into its number.
//
//sp:native TF2Econ_TranslateLoadoutSlotNameToIndex
func LoadoutSlotNameToIndex(name string) int32 { return cosmetics.LoadoutSlotNameToIndex(name) }

// ParticleAttributeList is every effect in that set. The caller owns it.
//
//sp:native TF2Econ_GetParticleAttributeList
func ParticleAttributeList(set ParticleSet) List { return cosmetics.ParticleAttributeList(set) }

// SetAttributeByDefIndex writes an attribute onto an entity, which is how the
// particle gets attached to the hat.
//
//sp:native TF2Attrib_SetByDefIndex
func SetAttributeByDefIndex(entity int32, attribute int32, value float32) {
	cosmetics.SetAttributeByDefIndex(entity, attribute, value)
}

// SetWearableAlwaysValid stops the game throwing out a wearable whose item it
// thinks the wearer does not own, which a bot never does.
//
//sp:native TF2Util_SetWearableAlwaysValid
func SetWearableAlwaysValid(wearable int32, valid bool) {
	cosmetics.SetWearableAlwaysValid(wearable, valid)
}

// RemoveWearable takes the hat off the way the game does, so the player's own
// list of wearables is updated with it.
//
//sp:native TF2_RemoveWearable
func RemoveWearable(client int32, wearable int32) { cosmetics.RemoveWearable(client, wearable) }

// SetEntityModel gives the entity a shape.
//
//sp:native SetEntityModel
func SetEntityModel(entity int32, model Text) { cosmetics.SetEntityModel(entity, model) }

// RemoveEntity deletes it.
//
//sp:native RemoveEntity
func RemoveEntity(entity int32) { cosmetics.RemoveEntity(entity) }

// IsModelPrecached says the map already loaded that model.
//
//sp:native IsModelPrecached
func IsModelPrecached(model Text) bool { return cosmetics.IsModelPrecached(model) }

// PrecacheModel adds it to the table, which anything has to be in before it can
// be worn.
//
//sp:native PrecacheModel
func PrecacheModel(model Text) int32 { return cosmetics.PrecacheModel(model) }

// Erase takes the entry at that index out and moves the rest down.
//
//sp:method Erase
func (l List) Erase(index int32) { cosmetics.ListErase(l, index) }

// FindValue is where that value is, or -1.
//
//sp:method FindValue
func (l List) FindValue(value int32) int32 { return cosmetics.ListFindValue(l, value) }

// BotHats is redbots_manager_bot_hats, whether the defenders wear anything.
//
//sp:global redbots_manager_bot_hats
func BotHats() ConVar { return 0 }

// BotHatEffects is redbots_manager_bot_hat_effects, whether what they wear
// carries a particle.
//
//sp:global redbots_manager_bot_hat_effects
func BotHatEffects() ConVar { return 0 }
