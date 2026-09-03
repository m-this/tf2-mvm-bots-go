package engine

/*
Calling a function the game has and SourceMod does not offer.

Each one is prepared once at load from a signature or a virtual table index the
gamedata file names, and the handle that comes back is what the wrapper calls.
A preparation that fails is counted rather than fatal: the mod says which ones
went missing and then stops, which is a better report than the first crash.
*/

// SDKCallCalls are the answers.
type SDKCallCalls struct {
	StartPrepSDKCall               func(kind int32)
	PrepSetFromConf                func(g GameData, source int32, name string) bool
	PrepAddParameter               func(kind int32, pass int32)
	PrepAddParameterFlagged        func(kind int32, pass int32, decodeFlags int32, encodeFlags int32)
	PrepSetReturnInfo              func(kind int32, pass int32)
	EndPrepSDKCall                 func() Call
	SetCall                        func(name string, c Call)
	GetCall                        func(name string) Call
	NewGameDataText                func(name Text) GameData
	PrepSetFromConfText            func(g GameData, source int32, name string) bool
	DoPostInventoryApplication     func(client int32)
	DoSetMission                   func(client int32, mission int32, resetBehaviorSystem bool)
	DoLookupBone                   func(entity int32, name string) int32
	DoHasAmmo                      func(weapon int32) bool
	DoClip1                        func(weapon int32) int32
	DoGetProjectileSpeed           func(weapon int32) float32
	DoGEconItemSchema              func() Address
	DoGetAttributeDefinitionByName func(schema Address, name string) Address
	DoCanUpgradeWithAttrib         func(player int32, slot int32, attrib int32, upgrade Address) bool
	DoGetCostForUpgrade            func(upgrade Address, slot int32, class int32, purchaser int32) int32
	DoGetUpgradeTier               func(upgrade int32) int32
	DoIsUpgradeTierEnabled         func(player int32, slot int32, upgrade int32) bool
	DoShouldCollide                func(entity int32, collisionGroup int32, contentsMask int32) bool
	DoGetBonePosition              func(entity int32, bone int32) ([3]float32, [3]float32)
	DoAimHeadTowards               func(body Body, lookAtPos [3]float32, priority LookAtPriority, duration float32, replyWhenAimed Address, reason string)
}

var sdkCalls SDKCallCalls

// InstallSDKCalls puts a set of answers behind them.
func InstallSDKCalls(c SDKCallCalls) func() {
	previous := sdkCalls
	sdkCalls = c
	return func() { sdkCalls = previous }
}

// Call is a prepared SDKCall, which the caller owns for the life of the plugin.
//
//sp:tag Handle
type Call int32

// NoCall is null, what a preparation that failed gives back.
//
//sp:global null
func NoCall() Call { return 0 }

// SdkCallPlayer is SDKCall_Player, a method on CBasePlayer.
//
//sp:global SDKCall_Player
func SdkCallPlayer() int32 { return 0 }

// SdkCallEntity is SDKCall_Entity, a method on CBaseEntity.
//
//sp:global SDKCall_Entity
func SdkCallEntity() int32 { return 0 }

// SdkCallRaw is SDKCall_Raw, a method on something that is not an entity: the
// nextbot interfaces are reached this way.
//
//sp:global SDKCall_Raw
func SdkCallRaw() int32 { return 0 }

// SdkCallStatic is SDKCall_Static, a free function.
//
//sp:global SDKCall_Static
func SdkCallStatic() int32 { return 0 }

// SdkCallGameRules is SDKCall_GameRules, a method on CTFGameRules.
//
//sp:global SDKCall_GameRules
func SdkCallGameRules() int32 { return 0 }

// SdkConfSignature is SDKConf_Signature, found by a byte pattern.
//
//sp:global SDKConf_Signature
func SdkConfSignature() int32 { return 0 }

// SdkConfVirtual is SDKConf_Virtual, found by a virtual table index.
//
//sp:global SDKConf_Virtual
func SdkConfVirtual() int32 { return 0 }

// SdkTypePlain is SDKType_PlainOldData, a cell.
//
//sp:global SDKType_PlainOldData
func SdkTypePlain() int32 { return 0 }

// SdkTypeBool is SDKType_Bool.
//
//sp:global SDKType_Bool
func SdkTypeBool() int32 { return 0 }

// SdkTypeString is SDKType_String.
//
//sp:global SDKType_String
func SdkTypeString() int32 { return 0 }

// SdkTypeVector is SDKType_Vector.
//
//sp:global SDKType_Vector
func SdkTypeVector() int32 { return 0 }

// SdkTypeQAngle is SDKType_QAngle.
//
//sp:global SDKType_QAngle
func SdkTypeQAngle() int32 { return 0 }

// SdkTypeFloat is SDKType_Float.
//
//sp:global SDKType_Float
func SdkTypeFloat() int32 { return 0 }

// SdkTypePlayer is SDKType_CBasePlayer.
//
//sp:global SDKType_CBasePlayer
func SdkTypePlayer() int32 { return 0 }

// SdkPassPlain is SDKPass_Plain.
//
//sp:global SDKPass_Plain
func SdkPassPlain() int32 { return 0 }

// SdkPassPointer is SDKPass_Pointer.
//
//sp:global SDKPass_Pointer
func SdkPassPointer() int32 { return 0 }

// SdkPassByRef is SDKPass_ByRef.
//
//sp:global SDKPass_ByRef
func SdkPassByRef() int32 { return 0 }

// SdkPassByValue is SDKPass_ByValue.
//
//sp:global SDKPass_ByValue
func SdkPassByValue() int32 { return 0 }

// VencodeCopyback is VENCODE_FLAG_COPYBACK, which writes a by-reference
// parameter back into the caller's variable.
//
//sp:global VENCODE_FLAG_COPYBACK
func VencodeCopyback() int32 { return 0 }

// StartPrepSDKCall begins one.
//
//sp:native StartPrepSDKCall
func StartPrepSDKCall(kind int32) {
	if sdkCalls.StartPrepSDKCall == nil {
		missing("StartPrepSDKCall")
	}
	sdkCalls.StartPrepSDKCall(kind)
}

// PrepSetFromConf says where in the binary it is.
//
//sp:native PrepSDKCall_SetFromConf
func PrepSetFromConf(g GameData, source int32, name string) bool {
	if sdkCalls.PrepSetFromConf == nil {
		missing("PrepSDKCall_SetFromConf")
	}
	return sdkCalls.PrepSetFromConf(g, source, name)
}

// PrepAddParameter describes the next argument.
//
//sp:native PrepSDKCall_AddParameter
func PrepAddParameter(kind int32, pass int32) {
	if sdkCalls.PrepAddParameter == nil {
		missing("PrepSDKCall_AddParameter")
	}
	sdkCalls.PrepAddParameter(kind, pass)
}

// PrepAddParameterFlagged is the same with the encode flags written out, which
// is how a by-reference vector is copied back.
//
//sp:native PrepSDKCall_AddParameter
func PrepAddParameterFlagged(kind int32, pass int32, decodeFlags int32, encodeFlags int32) {
	if sdkCalls.PrepAddParameterFlagged == nil {
		missing("PrepSDKCall_AddParameter")
	}
	sdkCalls.PrepAddParameterFlagged(kind, pass, decodeFlags, encodeFlags)
}

// PrepSetReturnInfo describes what comes back.
//
//sp:native PrepSDKCall_SetReturnInfo
func PrepSetReturnInfo(kind int32, pass int32) {
	if sdkCalls.PrepSetReturnInfo == nil {
		missing("PrepSDKCall_SetReturnInfo")
	}
	sdkCalls.PrepSetReturnInfo(kind, pass)
}

// EndPrepSDKCall finishes it, and answers null when the binary had no such
// function.
//
//sp:native EndPrepSDKCall
func EndPrepSDKCall() Call {
	if sdkCalls.EndPrepSDKCall == nil {
		missing("EndPrepSDKCall")
	}
	return sdkCalls.EndPrepSDKCall()
}

/*
Reading and writing the prepared handles.

One setter and one reader per handle, because the name is what SourcePawn writes
on either side of the assignment and there is nothing generic to hold.
*/

// SetCallPostInventoryApplication writes m_hPostInventoryApplication.
//
//sp:globalset m_hPostInventoryApplication
func SetCallPostInventoryApplication(c Call) { sdkCalls.set("m_hPostInventoryApplication", c) }

// CallPostInventoryApplication reads it.
//
//sp:global m_hPostInventoryApplication
func CallPostInventoryApplication() Call { return sdkCalls.get("m_hPostInventoryApplication") }

// SetCallSetMission writes m_hSetMission.
//
//sp:globalset m_hSetMission
func SetCallSetMission(c Call) { sdkCalls.set("m_hSetMission", c) }

// CallSetMission reads it.
//
//sp:global m_hSetMission
func CallSetMission() Call { return sdkCalls.get("m_hSetMission") }

// SetCallLookupBone writes m_hLookupBone.
//
//sp:globalset m_hLookupBone
func SetCallLookupBone(c Call) { sdkCalls.set("m_hLookupBone", c) }

// CallLookupBone reads it.
//
//sp:global m_hLookupBone
func CallLookupBone() Call { return sdkCalls.get("m_hLookupBone") }

// SetCallGetBonePosition writes m_hGetBonePosition.
//
//sp:globalset m_hGetBonePosition
func SetCallGetBonePosition(c Call) { sdkCalls.set("m_hGetBonePosition", c) }

// CallGetBonePosition reads it.
//
//sp:global m_hGetBonePosition
func CallGetBonePosition() Call { return sdkCalls.get("m_hGetBonePosition") }

// SetCallHasAmmo writes m_hHasAmmo.
//
//sp:globalset m_hHasAmmo
func SetCallHasAmmo(c Call) { sdkCalls.set("m_hHasAmmo", c) }

// CallHasAmmo reads it.
//
//sp:global m_hHasAmmo
func CallHasAmmo() Call { return sdkCalls.get("m_hHasAmmo") }

// SetCallClip1 writes m_hClip1.
//
//sp:globalset m_hClip1
func SetCallClip1(c Call) { sdkCalls.set("m_hClip1", c) }

// CallClip1 reads it.
//
//sp:global m_hClip1
func CallClip1() Call { return sdkCalls.get("m_hClip1") }

// SetCallGetProjectileSpeed writes m_hGetProjectileSpeed.
//
//sp:globalset m_hGetProjectileSpeed
func SetCallGetProjectileSpeed(c Call) { sdkCalls.set("m_hGetProjectileSpeed", c) }

// CallGetProjectileSpeed reads it.
//
//sp:global m_hGetProjectileSpeed
func CallGetProjectileSpeed() Call { return sdkCalls.get("m_hGetProjectileSpeed") }

// SetCallAimHeadTowards writes m_hAimHeadTowards.
//
//sp:globalset m_hAimHeadTowards
func SetCallAimHeadTowards(c Call) { sdkCalls.set("m_hAimHeadTowards", c) }

// CallAimHeadTowards reads it.
//
//sp:global m_hAimHeadTowards
func CallAimHeadTowards() Call { return sdkCalls.get("m_hAimHeadTowards") }

// SetCallGEconItemSchema writes m_hGEconItemSchema.
//
//sp:globalset m_hGEconItemSchema
func SetCallGEconItemSchema(c Call) { sdkCalls.set("m_hGEconItemSchema", c) }

// CallGEconItemSchema reads it.
//
//sp:global m_hGEconItemSchema
func CallGEconItemSchema() Call { return sdkCalls.get("m_hGEconItemSchema") }

// SetCallGetAttributeDefinitionByName writes m_hGetAttributeDefinitionByName.
//
//sp:globalset m_hGetAttributeDefinitionByName
func SetCallGetAttributeDefinitionByName(c Call) { sdkCalls.set("m_hGetAttributeDefinitionByName", c) }

// CallGetAttributeDefinitionByName reads it.
//
//sp:global m_hGetAttributeDefinitionByName
func CallGetAttributeDefinitionByName() Call { return sdkCalls.get("m_hGetAttributeDefinitionByName") }

// SetCallCanUpgradeWithAttrib writes m_hCanUpgradeWithAttrib.
//
//sp:globalset m_hCanUpgradeWithAttrib
func SetCallCanUpgradeWithAttrib(c Call) { sdkCalls.set("m_hCanUpgradeWithAttrib", c) }

// CallCanUpgradeWithAttrib reads it.
//
//sp:global m_hCanUpgradeWithAttrib
func CallCanUpgradeWithAttrib() Call { return sdkCalls.get("m_hCanUpgradeWithAttrib") }

// SetCallGetCostForUpgrade writes m_hGetCostForUpgrade.
//
//sp:globalset m_hGetCostForUpgrade
func SetCallGetCostForUpgrade(c Call) { sdkCalls.set("m_hGetCostForUpgrade", c) }

// CallGetCostForUpgrade reads it.
//
//sp:global m_hGetCostForUpgrade
func CallGetCostForUpgrade() Call { return sdkCalls.get("m_hGetCostForUpgrade") }

// SetCallGetUpgradeTier writes m_hGetUpgradeTier.
//
//sp:globalset m_hGetUpgradeTier
func SetCallGetUpgradeTier(c Call) { sdkCalls.set("m_hGetUpgradeTier", c) }

// CallGetUpgradeTier reads it.
//
//sp:global m_hGetUpgradeTier
func CallGetUpgradeTier() Call { return sdkCalls.get("m_hGetUpgradeTier") }

// SetCallIsUpgradeTierEnabled writes m_hIsUpgradeTierEnabled.
//
//sp:globalset m_hIsUpgradeTierEnabled
func SetCallIsUpgradeTierEnabled(c Call) { sdkCalls.set("m_hIsUpgradeTierEnabled", c) }

// CallIsUpgradeTierEnabled reads it.
//
//sp:global m_hIsUpgradeTierEnabled
func CallIsUpgradeTierEnabled() Call { return sdkCalls.get("m_hIsUpgradeTierEnabled") }

// SetCallShouldCollide writes m_hShouldCollide.
//
//sp:globalset m_hShouldCollide
func SetCallShouldCollide(c Call) { sdkCalls.set("m_hShouldCollide", c) }

// CallShouldCollide reads it.
//
//sp:global m_hShouldCollide
func CallShouldCollide() Call { return sdkCalls.get("m_hShouldCollide") }

// set records one, which is all a Go process can do with it.
func (c SDKCallCalls) set(name string, value Call) {
	if c.SetCall == nil {
		missing(name)
	}
	c.SetCall(name, value)
}

// get reads one back.
func (c SDKCallCalls) get(name string) Call {
	if c.GetCall == nil {
		missing(name)
	}
	return c.GetCall(name)
}

// NewGameDataText parses a gamedata file whose name is in a buffer.
//
//sp:new GameData
func NewGameDataText(name Text) GameData {
	if sdkCalls.NewGameDataText == nil {
		missing("new GameData")
	}
	return sdkCalls.NewGameDataText(name)
}

// PrepSetFromConfText is PrepSDKCall_SetFromConf against a config the caller
// opened rather than the mod's own.
//
//sp:native PrepSDKCall_SetFromConf
func PrepSetFromConfText(g GameData, source int32, name string) bool {
	if sdkCalls.PrepSetFromConfText == nil {
		missing("PrepSDKCall_SetFromConf")
	}
	return sdkCalls.PrepSetFromConfText(g, source, name)
}

/*
The call itself, one extern per prepared handle.

SDKCall takes the handle first, which is what before says, and the rest is the
function's own arguments.
*/

// DoPostInventoryApplication runs the prepared call.
//
//sp:native SDKCall before m_hPostInventoryApplication
func DoPostInventoryApplication(client int32) {
	if sdkCalls.DoPostInventoryApplication == nil {
		missing("SDKCall")
	}
	sdkCalls.DoPostInventoryApplication(client)
}

// DoSetMission runs the prepared call.
//
//sp:native SDKCall before m_hSetMission
func DoSetMission(client int32, mission int32, resetBehaviorSystem bool) {
	if sdkCalls.DoSetMission == nil {
		missing("SDKCall")
	}
	sdkCalls.DoSetMission(client, mission, resetBehaviorSystem)
}

// DoLookupBone runs the prepared call.
//
//sp:native SDKCall before m_hLookupBone
func DoLookupBone(entity int32, name string) int32 {
	if sdkCalls.DoLookupBone == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoLookupBone(entity, name)
}

// DoHasAmmo runs the prepared call.
//
//sp:native SDKCall before m_hHasAmmo
func DoHasAmmo(weapon int32) bool {
	if sdkCalls.DoHasAmmo == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoHasAmmo(weapon)
}

// DoClip1 runs the prepared call.
//
//sp:native SDKCall before m_hClip1
func DoClip1(weapon int32) int32 {
	if sdkCalls.DoClip1 == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoClip1(weapon)
}

// DoGetProjectileSpeed runs the prepared call.
//
//sp:native SDKCall before m_hGetProjectileSpeed
func DoGetProjectileSpeed(weapon int32) float32 {
	if sdkCalls.DoGetProjectileSpeed == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGetProjectileSpeed(weapon)
}

// DoGEconItemSchema runs the prepared call.
//
//sp:native SDKCall before m_hGEconItemSchema
func DoGEconItemSchema() Address {
	if sdkCalls.DoGEconItemSchema == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGEconItemSchema()
}

// DoGetAttributeDefinitionByName runs the prepared call.
//
//sp:native SDKCall before m_hGetAttributeDefinitionByName
func DoGetAttributeDefinitionByName(schema Address, name string) Address {
	if sdkCalls.DoGetAttributeDefinitionByName == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGetAttributeDefinitionByName(schema, name)
}

// DoCanUpgradeWithAttrib runs the prepared call.
//
//sp:native SDKCall before m_hCanUpgradeWithAttrib
func DoCanUpgradeWithAttrib(player int32, slot int32, attrib int32, upgrade Address) bool {
	if sdkCalls.DoCanUpgradeWithAttrib == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoCanUpgradeWithAttrib(player, slot, attrib, upgrade)
}

// DoGetCostForUpgrade runs the prepared call.
//
//sp:native SDKCall before m_hGetCostForUpgrade
func DoGetCostForUpgrade(upgrade Address, slot int32, class int32, purchaser int32) int32 {
	if sdkCalls.DoGetCostForUpgrade == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGetCostForUpgrade(upgrade, slot, class, purchaser)
}

// DoGetUpgradeTier runs the prepared call.
//
//sp:native SDKCall before m_hGetUpgradeTier
func DoGetUpgradeTier(upgrade int32) int32 {
	if sdkCalls.DoGetUpgradeTier == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGetUpgradeTier(upgrade)
}

// DoIsUpgradeTierEnabled runs the prepared call.
//
//sp:native SDKCall before m_hIsUpgradeTierEnabled
func DoIsUpgradeTierEnabled(player int32, slot int32, upgrade int32) bool {
	if sdkCalls.DoIsUpgradeTierEnabled == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoIsUpgradeTierEnabled(player, slot, upgrade)
}

// DoShouldCollide runs the prepared call.
//
//sp:native SDKCall before m_hShouldCollide
func DoShouldCollide(entity int32, collisionGroup int32, contentsMask int32) bool {
	if sdkCalls.DoShouldCollide == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoShouldCollide(entity, collisionGroup, contentsMask)
}

// DoGetBonePosition runs the prepared call, which writes both a place and an
// angle back through the caller's variables.
//
//sp:native SDKCall before m_hGetBonePosition
func DoGetBonePosition(entity int32, bone int32) (origin [3]float32, angles [3]float32) {
	if sdkCalls.DoGetBonePosition == nil {
		missing("SDKCall")
	}
	return sdkCalls.DoGetBonePosition(entity, bone)
}

// DoAimHeadTowards runs the prepared call.
//
//sp:native SDKCall before m_hAimHeadTowards
func DoAimHeadTowards(body Body, lookAtPos [3]float32, priority LookAtPriority, duration float32, replyWhenAimed Address, reason string) {
	if sdkCalls.DoAimHeadTowards == nil {
		missing("SDKCall")
	}
	sdkCalls.DoAimHeadTowards(body, lookAtPos, priority, duration, replyWhenAimed, reason)
}

// LookAtPriority is LookAtPriorityType, how hard the body should try to hold
// the aim it is given.
//
//sp:tag LookAtPriorityType
type LookAtPriority int32
