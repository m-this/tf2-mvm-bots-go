package engine

/*
Fighting: picking a target, keeping it, and moving while shooting it.

The locomotion interface is here because the strafe uses it directly: a step to
one side is tested before it is taken, since locomotion stops itself walking
into a wall and will happily walk off a ledge.
*/

// FightCalls are the answers.
type FightCalls struct {
	TFBotMission                   func(client int32) int32
	CanUsePrimaryWeapon            func(client int32) bool
	ClientEyePosition              func(client int32) [3]float32
	DesiredAttackRange             func(client int32) float32
	BotNearestToBombNearestToHatch func(client int32) int32
	SelectRandomReachableEnemy     func(actor int32) int32
	HealerOfPlayer                 func(client int32, playerOnly bool) int32
	LastKnownArea                  func(c Combat) NavArea
	IsOnGround                     func(l Locomotion) bool
	IsClimbingOrJumping            func(l Locomotion) bool
	IsPotentiallyTraversable       func(l Locomotion, from [3]float32, to [3]float32, when int32) bool
	HasPotentialGap                func(l Locomotion, from [3]float32, to [3]float32) bool
	Approach                       func(l Locomotion, goal [3]float32)
	CampBomb                       func() Behaviour
	GuardPoint                     func() Behaviour
	DestroyTeleporter              func() Behaviour
	CollectMoney                   func() Behaviour
	CampBombIsPossible             func(client int32) bool
	GuardPointIsPossible           func(client int32) bool
	CollectMoneyIsPossible         func(client int32) bool
	DestroyTeleporterSelectTarget  func(actor int32) bool
}

var fights FightCalls

// InstallFights puts a set of answers behind them.
func InstallFights(c FightCalls) func() {
	previous := fights
	fights = c
	return func() { fights = previous }
}

// MissionSniper is CTFBot_MISSION_SNIPER.
//
//sp:global CTFBot_MISSION_SNIPER
func MissionSniper() int32 { return 3 }

// FeatureAttackStrafe is the switch for sidestepping while shooting.
//
//sp:global FEATURE_ATTACK_STRAFE
func FeatureAttackStrafe() int32 { return 8 }

// Immediately is IMMEDIATELY, the traversability question asked about right now
// rather than eventually.
//
//sp:global IMMEDIATELY
func Immediately() int32 { return 0 }

// ClassSniper is TFClass_Sniper.
//
//sp:global TFClass_Sniper
func ClassSniper() Class { return 2 }

// ClassSpy is TFClass_Spy.
//
//sp:global TFClass_Spy
func ClassSpy() Class { return 8 }

// TFBotMission is what the game told the robot to do, which for a sniper is
// what makes him sit on a perch.
//
//sp:plugin GetTFBotMission
func TFBotMission(client int32) int32 {
	if fights.TFBotMission == nil {
		missing("GetTFBotMission")
	}
	return fights.TFBotMission(client)
}

// CanUsePrimaryWeapon says the sniper has his rifle back.
//
//sp:plugin CanUsePrimayWeapon
func CanUsePrimaryWeapon(client int32) bool {
	if fights.CanUsePrimaryWeapon == nil {
		missing("CanUsePrimayWeapon")
	}
	return fights.CanUsePrimaryWeapon(client)
}

// ClientEyePosition is where the bot is looking from.
//
//sp:native GetClientEyePosition
func ClientEyePosition(client int32) (position [3]float32) {
	if fights.ClientEyePosition == nil {
		missing("GetClientEyePosition")
	}
	return fights.ClientEyePosition(client)
}

// DesiredAttackRange is how close this class wants to be.
//
//sp:plugin GetDesiredAttackRange
func DesiredAttackRange(client int32) float32 {
	if fights.DesiredAttackRange == nil {
		missing("GetDesiredAttackRange")
	}
	return fights.DesiredAttackRange(client)
}

// BotNearestToBombNearestToHatch is the robot worth shooting: the one carrying
// the bomb, or nearest to whoever is.
//
//sp:body FindBotNearestToBombNearestToHatch
func BotNearestToBombNearestToHatch(client int32) int32 {
	if fights.BotNearestToBombNearestToHatch == nil {
		missing("FindBotNearestToBombNearestToHatch")
	}
	return fights.BotNearestToBombNearestToHatch(client)
}

// SelectRandomReachableEnemy is anybody the bot could actually get to.
//
//sp:body SelectRandomReachableEnemy
func SelectRandomReachableEnemy(actor int32) int32 {
	if fights.SelectRandomReachableEnemy == nil {
		missing("SelectRandomReachableEnemy")
	}
	return fights.SelectRandomReachableEnemy(actor)
}

// HealerOfPlayer is whoever is keeping him alive, which is worth shooting first.
//
//sp:plugin GetHealerOfPlayer
func HealerOfPlayer(client int32, playerOnly bool) int32 {
	if fights.HealerOfPlayer == nil {
		missing("GetHealerOfPlayer")
	}
	return fights.HealerOfPlayer(client, playerOnly)
}

// LastKnownArea is the ground the game last saw him on.
//
//sp:method GetLastKnownArea
func (c Combat) LastKnownArea() NavArea {
	if fights.LastKnownArea == nil {
		missing("CBaseCombatCharacter.GetLastKnownArea")
	}
	return fights.LastKnownArea(c)
}

// IsOnGround says the bot has its feet down.
//
//sp:method IsOnGround
func (l Locomotion) IsOnGround() bool {
	if fights.IsOnGround == nil {
		missing("ILocomotion.IsOnGround")
	}
	return fights.IsOnGround(l)
}

// IsClimbingOrJumping says it is in the middle of something.
//
//sp:method IsClimbingOrJumping
func (l Locomotion) IsClimbingOrJumping() bool {
	if fights.IsClimbingOrJumping == nil {
		missing("ILocomotion.IsClimbingOrJumping")
	}
	return fights.IsClimbingOrJumping(l)
}

// IsPotentiallyTraversable says the step would not walk into anything.
//
//sp:method IsPotentiallyTraversable
func (l Locomotion) IsPotentiallyTraversable(from [3]float32, to [3]float32, when int32) bool {
	if fights.IsPotentiallyTraversable == nil {
		missing("ILocomotion.IsPotentiallyTraversable")
	}
	return fights.IsPotentiallyTraversable(l, from, to, when)
}

// HasPotentialGap says the step would walk off something.
//
//sp:method HasPotentialGap
func (l Locomotion) HasPotentialGap(from [3]float32, to [3]float32) bool {
	if fights.HasPotentialGap == nil {
		missing("ILocomotion.HasPotentialGap")
	}
	return fights.HasPotentialGap(l, from, to)
}

// Approach takes one step towards a point, which is a step and not a journey.
//
//sp:method Approach
func (l Locomotion) Approach(goal [3]float32) {
	if fights.Approach == nil {
		missing("ILocomotion.Approach")
	}
	fights.Approach(l, goal)
}

// The behaviours this one hands off to. All four are ported.

// CampBomb is CTFBotCampBomb.
//
//sp:body CTFBotCampBomb
func CampBomb() Behaviour {
	if fights.CampBomb == nil {
		missing("CTFBotCampBomb")
	}
	return fights.CampBomb()
}

// GuardPoint is CTFBotGuardPoint.
//
//sp:body CTFBotGuardPoint
func GuardPoint() Behaviour {
	if fights.GuardPoint == nil {
		missing("CTFBotGuardPoint")
	}
	return fights.GuardPoint()
}

// DestroyTeleporter is CTFBotDestroyTeleporter.
//
//sp:body CTFBotDestroyTeleporter
func DestroyTeleporter() Behaviour {
	if fights.DestroyTeleporter == nil {
		missing("CTFBotDestroyTeleporter")
	}
	return fights.DestroyTeleporter()
}

// CollectMoney is CTFBotCollectMoney.
//
//sp:body CTFBotCollectMoney
func CollectMoney() Behaviour {
	if fights.CollectMoney == nil {
		missing("CTFBotCollectMoney")
	}
	return fights.CollectMoney()
}

// CampBombIsPossible is that behaviour's precondition.
//
//sp:body CTFBotCampBomb_IsPossible
func CampBombIsPossible(client int32) bool {
	if fights.CampBombIsPossible == nil {
		missing("CTFBotCampBomb_IsPossible")
	}
	return fights.CampBombIsPossible(client)
}

// GuardPointIsPossible is that one's.
//
//sp:body CTFBotGuardPoint_IsPossible
func GuardPointIsPossible(client int32) bool {
	if fights.GuardPointIsPossible == nil {
		missing("CTFBotGuardPoint_IsPossible")
	}
	return fights.GuardPointIsPossible(client)
}

// CollectMoneyIsPossible is that one's.
//
//sp:body CTFBotCollectMoney_IsPossible
func CollectMoneyIsPossible(client int32) bool {
	if fights.CollectMoneyIsPossible == nil {
		missing("CTFBotCollectMoney_IsPossible")
	}
	return fights.CollectMoneyIsPossible(client)
}

// DestroyTeleporterSelectTarget is that one's.
//
//sp:body CTFBotDestroyTeleporter_SelectTarget
func DestroyTeleporterSelectTarget(actor int32) bool {
	if fights.DestroyTeleporterSelectTarget == nil {
		missing("CTFBotDestroyTeleporter_SelectTarget")
	}
	return fights.DestroyTeleporterSelectTarget(actor)
}
