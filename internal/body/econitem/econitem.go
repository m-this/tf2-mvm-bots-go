/*
Package econitem is the part of source/redbots3/util.sp that builds a weapon out
of nothing and puts it in a bot's hands.

The game will not create one from an item definition alone: the quality and the
level are written at offsets SetEntProp refuses, and a builder needs its object
table filled in or the client crashes reading garbage.
*/
package econitem

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The offsets found once and kept, which the shipped file holds in static locals.
var (
	//sp:name iOffsetEntityQuality
	offsetEntityQuality int32 = -1
	//sp:name iOffsetEntityLevel
	offsetEntityLevel int32 = -1
)

// CreateNoSpawn makes the item without bringing it into the world, so the
// caller can finish it first.
//
//sp:name EconItemCreateNoSpawn
//sp:writable classname
func CreateNoSpawn(classname string, itemDefIndex int32, level int32, quality int32) int32 {
	item := engine.CreateEntityByName(classname)

	if item != -1 {
		engine.SetEntPropSend(item, engine.PropSend(), "m_iItemDefinitionIndex", itemDefIndex)
		engine.SetEntPropSend(item, engine.PropSend(), "m_bInitialized", 1)

		// SetEntProp doesn't work here...
		if offsetEntityQuality == -1 {
			offsetEntityQuality = engine.FindSendPropInfo("CEconEntity", "m_iEntityQuality")
		}

		if offsetEntityLevel == -1 {
			offsetEntityLevel = engine.FindSendPropInfo("CEconEntity", "m_iEntityLevel")
		}

		engine.SetEntData(item, offsetEntityQuality, quality)
		engine.SetEntData(item, offsetEntityLevel, level)

		if engine.StrEqualLiteral(classname, "tf_weapon_builder", false) {
			/* NOTE: After the 2023-10-09 update, not setting netprop m_iObjectType
			will crash all client games (but the server will remain fine)
			I suspect the client's game code change and not setting it cause it to read garbage */
			engine.SetEntPropSend(item, engine.PropSend(), "m_iObjectType", 3) // Set to OBJ_ATTACHMENT_SAPPER?

			isSapper := engine.IsItemDefIndexSapper(itemDefIndex)

			if isSapper {
				engine.SetEntPropSend(item, engine.PropData(), "m_iSubType", 3)
			}

			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", engine.ChooseInt(isSapper, 0, 1), 4, 0) // OBJ_DISPENSER
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", engine.ChooseInt(isSapper, 0, 1), 4, 1) // OBJ_TELEPORTER
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", engine.ChooseInt(isSapper, 0, 1), 4, 2) // OBJ_SENTRYGUN
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", engine.ChooseInt(isSapper, 1, 0), 4, 3) // OBJ_ATTACHMENT_SAPPER
		} else if engine.StrEqualLiteral(classname, "tf_weapon_sapper", false) {
			engine.SetEntPropSend(item, engine.PropSend(), "m_iObjectType", 3)
			engine.SetEntPropSend(item, engine.PropData(), "m_iSubType", 3)
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", 0, 4, 0)
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", 0, 4, 1)
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", 0, 4, 2)
			engine.SetEntPropAt(item, engine.PropSend(), "m_aBuildableObjectTypes", 1, 4, 3)
		}
	} else {
		engine.LogError("EconItemCreateNoSpawn: Failed to create entity.")
	}

	return item
}

// SpawnGiveTo brings it into the world and equips it. Call this when you
// are ready to spawn it.
//
//sp:name EconItemSpawnGiveTo
func SpawnGiveTo(item int32, client int32) {
	engine.DispatchSpawn(item)

	if engine.IsEntityWearable(item) {
		engine.EquipPlayerWearable(client, item)
	} else {
		engine.EquipPlayerWeapon(client, item)
	}

	// NOTE: bot items are always visible in PvE, so m_bValidatedAttachedEntity does not need setting
}

// GiveItemToPlayer is both halves at once, which is what every caller wants.
//
//sp:name GiveItemToPlayer
//sp:writable classname
func GiveItemToPlayer(client int32, classname string, itemDefIndex int32, level int32, quality int32) int32 {
	item := CreateNoSpawn(classname, itemDefIndex, level, quality)

	if item != -1 {
		engine.SetItemID(item, engine.RandomInt(1, 2048))
		SpawnGiveTo(item, client)
	}

	return item
}
