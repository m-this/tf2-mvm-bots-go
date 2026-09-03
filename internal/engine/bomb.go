package engine

/*
The bomb, and the ground a bot holds around it.

Most of this is the plugin's own and goes when the port reaches it. The three
action constructors are here because a behaviour may hand the engine another
one, and the ones it hands over have not been ported yet.
*/

// BombCalls are the answers.
type BombCalls struct {
	CaptureFlagIsHome            func(flag int32) bool
	BombNearestToHatch           func() int32
	OwnerEntity                  func(entity int32) int32
	EquipBestWeaponForThreat     func(client int32, threat Known)
	IsLineOfFireClearPosition    func(client int32, from [3]float32, to [3]float32) bool
	WantsDispenser               func(client int32) bool
	FindFriendlyDispenserNear    func(client int32, origin [3]float32) int32
	Feature                      func(id int32) bool
	AttackTank                   func() Behaviour
	DefenderAttack               func() Behaviour
	AttackTankSelectTarget       func(client int32) bool
	EnemyPlayerNearestToPosition func(client int32, position [3]float32, maxDistance float32) int32
}

var bombs BombCalls

// InstallBombs puts a set of answers behind them.
func InstallBombs(c BombCalls) func() {
	previous := bombs
	Fill(&c)
	bombs = c
	return func() { bombs = previous }
}

// ConceptSentryHere is MP_CONCEPT_PLAYER_SENTRYHERE, which is the bot telling
// the team where it is holding.
//
//sp:global MP_CONCEPT_PLAYER_SENTRYHERE
func ConceptSentryHere() int32 { return 1 }

// FeatureDispenserGuard is the switch for holding from a dispenser.
//
//sp:global FEATURE_DISPENSER_GUARD
func FeatureDispenserGuard() int32 { return 1 }

// Feature says whether a switch is on.
//
//sp:plugin Feature
func Feature(id int32) bool { return bombs.Feature(id) }

// BombNearestToHatch is the bomb worth holding, and -1 when there is none.
//
//sp:body FindBombNearestToHatch
func BombNearestToHatch() int32 { return bombs.BombNearestToHatch() }

// OwnerEntity is who is carrying it, and -1 for nobody.
//
//sp:native BaseEntity_GetOwnerEntity
func OwnerEntity(entity int32) int32 { return bombs.OwnerEntity(entity) }

// EquipBestWeaponForThreat puts the right thing in the bot's hands.
//
//sp:body EquipBestWeaponForThreat
func EquipBestWeaponForThreat(client int32, threat Known) {
	bombs.EquipBestWeaponForThreat(client, threat)
}

// IsLineOfFireClearPosition says whether the bot could shoot at that spot.
//
//sp:body IsLineOfFireClearPosition
func IsLineOfFireClearPosition(client int32, from [3]float32, to [3]float32) bool {
	return bombs.IsLineOfFireClearPosition(client, from, to)
}

// WantsDispenser says whether the bot would use one if it were there.
//
//sp:body WantsDispenser
func WantsDispenser(client int32) bool { return bombs.WantsDispenser(client) }

// FindFriendlyDispenserNear is one on this ground, and -1 for none.
//
//sp:body FindFriendlyDispenserNear
func FindFriendlyDispenserNear(client int32, origin [3]float32) int32 {
	return bombs.FindFriendlyDispenserNear(client, origin)
}

// The behaviours a bot may be handed instead of this one. Not ported yet, so
// their constructors are reached rather than called.

// AttackTank is CTFBotAttackTank.
//
//sp:body CTFBotAttackTank
func AttackTank() Behaviour { return bombs.AttackTank() }

// DefenderAttack is CTFBotDefenderAttack.
//
//sp:body CTFBotDefenderAttack
func DefenderAttack() Behaviour { return bombs.DefenderAttack() }

// AttackTankSelectTarget is the tank behaviour's own precondition.
//
//sp:body CTFBotAttackTank_SelectTarget
func AttackTankSelectTarget(client int32) bool { return bombs.AttackTankSelectTarget(client) }

// EnemyPlayerNearestToPosition is util.sp:1550, ported.
//
//sp:body GetEnemyPlayerNearestToPosition
func EnemyPlayerNearestToPosition(client int32, position [3]float32, maxDistance float32) int32 {
	return bombs.EnemyPlayerNearestToPosition(client, position, maxDistance)
}

// ClassScout is TFClass_Scout.
//
//sp:global TFClass_Scout
func ClassScout() Class { return 1 }

// ClassSoldier is TFClass_Soldier.
//
//sp:global TFClass_Soldier
func ClassSoldier() Class { return 3 }

// ClassPyro is TFClass_Pyro.
//
//sp:global TFClass_Pyro
func ClassPyro() Class { return 7 }

// ClassDemoMan is TFClass_DemoMan.
//
//sp:global TFClass_DemoMan
func ClassDemoMan() Class { return 4 }

// CaptureFlagIsHome says the bomb is still on its stand, which is a bomb nobody
// is carrying anywhere.
//
//sp:library CaptureFlag_IsHome
func CaptureFlagIsHome(flag int32) bool { return bombs.CaptureFlagIsHome(flag) }
