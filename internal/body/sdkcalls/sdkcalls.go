/*
Package sdkcalls is source/redbots3/sdkcalls.sp: calling a function the game has
and SourceMod does not offer.

Each one is prepared once at load from a signature or a virtual table index the
gamedata file names, and the handle that comes back is what the wrapper calls.
A preparation that fails is counted rather than fatal: the mod says which ones
went missing and then stops, which is a better report than the first crash.
*/
package sdkcalls

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// InitSDKCalls prepares every one of them, and says whether all of them worked.
//
//sp:name InitSDKCalls
func InitSDKCalls(hGamedata engine.GameData) bool {
	iFailCount := int32(0)

	engine.StartPrepSDKCall(engine.SdkCallPlayer())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFPlayer::PostInventoryApplication")
	engine.SetCallPostInventoryApplication(engine.EndPrepSDKCall())
	if engine.CallPostInventoryApplication() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFPlayer::PostInventoryApplication!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallPlayer())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFBot::SetMission")
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypeBool(), engine.SdkPassPlain())
	engine.SetCallSetMission(engine.EndPrepSDKCall())
	if engine.CallSetMission() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFBot::SetMission!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CBaseAnimating::LookupBone")
	engine.PrepAddParameter(engine.SdkTypeString(), engine.SdkPassPointer())
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallLookupBone(engine.EndPrepSDKCall())
	if engine.CallLookupBone() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CBaseAnimating::LookupBone!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CBaseAnimating::GetBonePosition")
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameterFlagged(engine.SdkTypeVector(), engine.SdkPassByRef(), 0, engine.VencodeCopyback())
	engine.PrepAddParameterFlagged(engine.SdkTypeQAngle(), engine.SdkPassByRef(), 0, engine.VencodeCopyback())
	engine.SetCallGetBonePosition(engine.EndPrepSDKCall())
	if engine.CallGetBonePosition() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CBaseAnimating::GetBonePosition!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfVirtual(), "CBaseCombatWeapon::HasAmmo")
	engine.PrepSetReturnInfo(engine.SdkTypeBool(), engine.SdkPassByValue())
	engine.SetCallHasAmmo(engine.EndPrepSDKCall())
	if engine.CallHasAmmo() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CBaseCombatWeapon::HasAmmo!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfVirtual(), "CBaseCombatWeapon::Clip1")
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallClip1(engine.EndPrepSDKCall())
	if engine.CallClip1() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CBaseCombatWeapon::Clip1!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfVirtual(), "CTFWeaponBaseGun::GetProjectileSpeed")
	engine.PrepSetReturnInfo(engine.SdkTypeFloat(), engine.SdkPassPlain())
	engine.SetCallGetProjectileSpeed(engine.EndPrepSDKCall())
	if engine.CallGetProjectileSpeed() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFWeaponBaseGun::GetProjectileSpeed!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallRaw())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfVirtual(), "IBody::AimHeadTowards")
	engine.PrepAddParameter(engine.SdkTypeVector(), engine.SdkPassByRef())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypeFloat(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypeString(), engine.SdkPassPointer())
	engine.SetCallAimHeadTowards(engine.EndPrepSDKCall())
	if engine.CallAimHeadTowards() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for IBody::AimHeadTowards!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallStatic())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "GEconItemSchema")
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallGEconItemSchema(engine.EndPrepSDKCall())
	if engine.CallGEconItemSchema() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for GEconItemSchema!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallRaw())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CEconItemSchema::GetAttributeDefinitionByName")
	engine.PrepAddParameter(engine.SdkTypeString(), engine.SdkPassPointer())
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallGetAttributeDefinitionByName(engine.EndPrepSDKCall())
	if engine.CallGetAttributeDefinitionByName() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CEconItemSchema::GetAttributeDefinitionByName!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallGameRules())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFGameRules::CanUpgradeWithAttrib")
	engine.PrepAddParameter(engine.SdkTypePlayer(), engine.SdkPassPointer())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepSetReturnInfo(engine.SdkTypeBool(), engine.SdkPassByValue())
	engine.SetCallCanUpgradeWithAttrib(engine.EndPrepSDKCall())
	if engine.CallCanUpgradeWithAttrib() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFGameRules::CanUpgradeWithAttrib!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallGameRules())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFGameRules::GetCostForUpgrade")
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlayer(), engine.SdkPassPointer())
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallGetCostForUpgrade(engine.EndPrepSDKCall())
	if engine.CallGetCostForUpgrade() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFGameRules::GetCostForUpgrade!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallGameRules())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFGameRules::GetUpgradeTier")
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepSetReturnInfo(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.SetCallGetUpgradeTier(engine.EndPrepSDKCall())
	if engine.CallGetUpgradeTier() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFGameRules::GetUpgradeTier!")
		iFailCount++
	}

	engine.StartPrepSDKCall(engine.SdkCallGameRules())
	engine.PrepSetFromConf(hGamedata, engine.SdkConfSignature(), "CTFGameRules::IsUpgradeTierEnabled")
	engine.PrepAddParameter(engine.SdkTypePlayer(), engine.SdkPassPointer())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepSetReturnInfo(engine.SdkTypeBool(), engine.SdkPassByValue())
	engine.SetCallIsUpgradeTierEnabled(engine.EndPrepSDKCall())
	if engine.CallIsUpgradeTierEnabled() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CTFGameRules::IsUpgradeTierEnabled!")
		iFailCount++
	}

	// SDKHooks gamedata.
	sTempConfFileName := engine.LiteralText("sdkhooks.games/engine.ep2v")
	hTempConf := engine.NewGameDataText(sTempConfFileName)

	engine.StartPrepSDKCall(engine.SdkCallEntity())
	engine.PrepSetFromConfText(hTempConf, engine.SdkConfVirtual(), "ShouldCollide")
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepAddParameter(engine.SdkTypePlain(), engine.SdkPassPlain())
	engine.PrepSetReturnInfo(engine.SdkTypeBool(), engine.SdkPassByValue())
	engine.SetCallShouldCollide(engine.EndPrepSDKCall())
	if engine.CallShouldCollide() == engine.NoCall() {
		engine.LogError("Failed to create SDKCall for CBaseEntity::ShouldCollide from file %s.txt", sTempConfFileName)
		iFailCount++
	}

	hTempConf.Close()

	if iFailCount > 0 {
		engine.LogError("InitSDKCalls: GameData file has %d problems!", iFailCount)
		return false
	}

	return true
}

// PostInventoryApplication is the prepared call by that name.
//
//sp:name PostInventoryApplication
func PostInventoryApplication(client int32) {
	engine.DoPostInventoryApplication(client)
}

// SetMission is the prepared call by that name.
//
//sp:name SetMission
//sp:default resetBehaviorSystem true
func SetMission(client int32, mission int32, resetBehaviorSystem bool) {
	engine.DoSetMission(client, mission, resetBehaviorSystem)
}

// LookupBone is the prepared call by that name.
//
//sp:name LookupBone
func LookupBone(entity int32, szName string) int32 {
	return engine.DoLookupBone(entity, szName)
}

// HasAmmo is the prepared call by that name.
//
//sp:name HasAmmo
func HasAmmo(weapon int32) bool {
	return engine.DoHasAmmo(weapon)
}

// Clip1 is the prepared call by that name.
//
//sp:name Clip1
func Clip1(weapon int32) int32 {
	return engine.DoClip1(weapon)
}

// GetProjectileSpeed is the prepared call by that name.
//
//sp:name GetProjectileSpeed
func GetProjectileSpeed(weapon int32) float32 {
	return engine.DoGetProjectileSpeed(weapon)
}

// GEconItemSchema is the prepared call by that name.
//
//sp:name GEconItemSchema
func GEconItemSchema() engine.Address {
	return engine.DoGEconItemSchema()
}

// GetAttributeDefinitionByName is the prepared call by that name.
//
//sp:name GetAttributeDefinitionByName
func GetAttributeDefinitionByName(econItemSchema engine.Address, pszDefName string) engine.Address {
	return engine.DoGetAttributeDefinitionByName(econItemSchema, pszDefName)
}

// CanUpgradeWithAttrib is the prepared call by that name.
//
//sp:name CanUpgradeWithAttrib
func CanUpgradeWithAttrib(pPlayer int32, iWeaponSlot int32, iAttribIndex int32, pUpgrade engine.Address) bool {
	return engine.DoCanUpgradeWithAttrib(pPlayer, iWeaponSlot, iAttribIndex, pUpgrade)
}

// GetCostForUpgrade is the prepared call by that name.
//
//sp:name GetCostForUpgrade
//sp:default pPurchaser -1
func GetCostForUpgrade(pUpgrade engine.Address, iItemSlot int32, nClass int32, pPurchaser int32) int32 {
	return engine.DoGetCostForUpgrade(pUpgrade, iItemSlot, nClass, pPurchaser)
}

// GetUpgradeTier is the prepared call by that name.
//
//sp:name GetUpgradeTier
func GetUpgradeTier(iUpgrade int32) int32 {
	return engine.DoGetUpgradeTier(iUpgrade)
}

// IsUpgradeTierEnabled is the prepared call by that name.
//
//sp:name IsUpgradeTierEnabled
func IsUpgradeTierEnabled(pTFPlayer int32, iItemSlot int32, iUpgrade int32) bool {
	return engine.DoIsUpgradeTierEnabled(pTFPlayer, iItemSlot, iUpgrade)
}

// ShouldCollide is the prepared call by that name.
//
//sp:name ShouldCollide
func ShouldCollide(entity int32, collisionGroup int32, contentsMask int32) bool {
	return engine.DoShouldCollide(entity, collisionGroup, contentsMask)
}

// GetBonePosition is where that bone is and how it is turned.
//
//sp:name GetBonePosition
func GetBonePosition(entity int32, iBone int32) (origin [3]float32, angles [3]float32) {
	origin, angles = engine.DoGetBonePosition(entity, iBone)

	return origin, angles
}

// AimHeadTowards points the bot's head at a place, for a reason it can say.
//
//sp:name AimHeadTowards
//sp:default priority BORING
//sp:default duration 0.0
//sp:default replyWhenAimed Address_Null
//sp:default reason NULL_STRING
//sp:const reason
func AimHeadTowards(body engine.Body, lookAtPos [3]float32, priority engine.LookAtPriority, duration float32, replyWhenAimed engine.Address, reason string) {
	engine.DoAimHeadTowards(body, lookAtPos, priority, duration, replyWhenAimed, reason)
}
