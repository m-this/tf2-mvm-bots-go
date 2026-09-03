package engine

/*
What the spy behaviour reaches that the port has not written yet.

Every one of these is a plugin function or a SourceMod native the spy sap
behaviour calls, declared so that behaviour can be ported before the files
around it are. The //sp:plugin ones are the work still to do; internal/body
refuses to have both a plugin extern and a body of the same name, so each goes
the day the port reaches it.
*/

// SpyCalls are the answers the spy behaviour gets.
type SpyCalls struct {
	IsValidEntity              func(entity int32) bool
	IsBaseObject               func(entity int32) bool
	IsFeignDeathReady          func(client int32) bool
	PlayerWeaponSlot           func(client int32, slot int32) int32
	SetPlayerActiveWeapon      func(client int32, weapon int32)
	SetActiveWeapon            func(client int32, weapon int32) bool
	PressAltFireButton         func(client int32)
	PressFireButton            func(client int32)
	SnapViewToPosition         func(client int32, position [3]float32)
	UpdateLookAroundForEnemies func(client int32, allow bool)
	RepathToTarget             func(actor int32, bot Bot, target int32)
	UpdateLastKnownArea        func(entity int32)
	IsRangeLessThan            func(bot Bot, target int32, distance float32) bool
	GameTime                   func() float32
	RandomFloat                func(low float32, high float32) float32
	DesiredPathLookAheadRange  func(client int32) float32
	NearestSappableObject      func(client int32, maxDistance float32) int32
}

var spy SpyCalls

// InstallSpy puts a set of answers behind them.
func InstallSpy(c SpyCalls) func() {
	previous := spy
	Fill(&c)
	spy = c
	return func() { spy = previous }
}

// WeaponSlotSecondary is TFWeaponSlot_Secondary, where the sapper lives.
//
//sp:global TFWeaponSlot_Secondary
func WeaponSlotSecondary() int32 { return 1 }

// IsValidEntity says whether the index still refers to something.
//
//sp:native IsValidEntity
func IsValidEntity(entity int32) bool { return spy.IsValidEntity(entity) }

// IsBaseObject says whether the entity is a building.
//
//sp:native BaseEntity_IsBaseObject
func IsBaseObject(entity int32) bool { return spy.IsBaseObject(entity) }

// IsFeignDeathReady says whether the spy is holding a dead ringer.
//
//sp:native TF2_IsFeignDeathReady
func IsFeignDeathReady(client int32) bool { return spy.IsFeignDeathReady(client) }

// PlayerWeaponSlot is what the player has in that slot, or -1.
//
//sp:native GetPlayerWeaponSlot
func PlayerWeaponSlot(client int32, slot int32) int32 { return spy.PlayerWeaponSlot(client, slot) }

// SetActiveWeapon puts a weapon in the player's hands and says whether the game
// allowed it, which is what EquipWeaponSlot passes on to its caller.
//
//sp:native TF2Util_SetPlayerActiveWeapon
func SetActiveWeapon(client int32, weapon int32) bool { return spy.SetActiveWeapon(client, weapon) }

// SetPlayerActiveWeapon is the same native where the answer is not read.
//
//sp:native TF2Util_SetPlayerActiveWeapon
func SetPlayerActiveWeapon(client int32, weapon int32) { spy.SetPlayerActiveWeapon(client, weapon) }

// PressAltFireButton is how a bot uncloaks.
//
//sp:library VS_PressAltFireButton
func PressAltFireButton(client int32) { spy.PressAltFireButton(client) }

// PressFireButton is how a bot plants the sapper.
//
//sp:library VS_PressFireButton
func PressFireButton(client int32) { spy.PressFireButton(client) }

// SnapViewToPosition turns the bot to look at a point.
//
//sp:body SnapViewToPosition
func SnapViewToPosition(client int32, position [3]float32) { spy.SnapViewToPosition(client, position) }

// UpdateLookAroundForEnemies turns the bot's own looking back on or off, so a
// behaviour that aims for itself is not fought by the game. Ported, botqueries.
//
//sp:body UpdateLookAroundForEnemies
func UpdateLookAroundForEnemies(client int32, allow bool) {
	spy.UpdateLookAroundForEnemies(client, allow)
}

// RepathToTarget asks the game for a route to an entity.
//
//sp:body RepathToTarget
func RepathToTarget(actor int32, bot Bot, target int32) { spy.RepathToTarget(actor, bot, target) }

// UpdateLastKnownArea tells the target's nav area it was just seen, which the
// route needs before it is asked for.
//
//sp:method UpdateLastKnownArea
func (c Combat) UpdateLastKnownArea() { spy.UpdateLastKnownArea(int32(c)) }

// Combat is CBaseCombatCharacter, anything that fights.
//
//sp:tag CBaseCombatCharacter
type Combat int32

// CombatOf is the fighting side of an entity.
//
//sp:native CBaseCombatCharacter
func CombatOf(entity int32) Combat { return Combat(entity) }

// IsRangeLessThan says whether the bot is closer than that to the entity.
//
//sp:method IsRangeLessThan
func (b Bot) IsRangeLessThan(target int32, distance float32) bool {
	return spy.IsRangeLessThan(b, target, distance)
}

// GameTime is the server's clock.
//
//sp:native GetGameTime
func GameTime() float32 { return spy.GameTime() }

// RandomFloat is the game's own randomness, which mvm-z83.18 says has to come
// in as a parameter before any of this can be compared. It has not yet.
//
//sp:native GetRandomFloat
func RandomFloat(low float32, high float32) float32 { return spy.RandomFloat(low, high) }

// DesiredPathLookAheadRange is how far along the path a bot of that class aims.
// Ported, botqueries.
//
//sp:body GetDesiredPathLookAheadRange
func DesiredPathLookAheadRange(client int32) float32 { return spy.DesiredPathLookAheadRange(client) }

/*
The two below are the other direction from a plugin extern: internal/body/scan
generates both, so these say the port owns them and internal/body refuses the
declaration if that stops being true. The emitted SourcePawn is one flat
namespace, so calling them is calling them by name.
*/

// WorldSpaceCenter is the middle of the entity. Ported, util.sp:348.
//
//sp:body WorldSpaceCenter returns
func WorldSpaceCenter(entity int32) [3]float32 { return installed.WorldSpaceCenter(entity) }

// NearestSappableObject is the closest enemy building. Ported, util.sp:1325.
//
//sp:body GetNearestSappableObject
func NearestSappableObject(client int32, maxDistance float32) int32 {
	return spy.NearestSappableObject(client, maxDistance)
}
