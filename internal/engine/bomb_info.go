package engine

/*
The bomb, as a record.

BombInfo_t is an enum struct the plugin fills, and its fields are named the way
SourcePawn names them: vPosition, not Position. Go cannot export a field that
starts with a lower case letter, so the Go field carries the SourcePawn name in
a struct tag, which is what struct tags are for.
*/

// BombInfoCalls are the answers.
type BombInfoCalls struct {
	GetBombInfo                func() (bool, BombInfo)
	TravelDistanceToBombTarget func(area Area) float32
	IsBaseBoss                 func(entity int32) bool
	BombHatchPosition          func() [3]float32
	IsHeadAimingOnTarget       func(b Body) bool
}

var bombInfos BombInfoCalls

// InstallBombInfo puts a set of answers behind them.
func InstallBombInfo(c BombInfoCalls) func() {
	previous := bombInfos
	bombInfos = c
	return func() { bombInfos = previous }
}

// BombInfo is where the bomb is and what it is doing.
//
//sp:tag BombInfo_t
type BombInfo struct {
	Position [3]float32 `sp:"vPosition"`
	// MaxBattleFront is how far along the route the fighting has reached,
	// which is what a nest is compared against before it moves forward.
	MaxBattleFront float32 `sp:"flMaxBattleFront"`
}

// PropData is Prop_Data, the datamap table rather than the networked one.
//
//sp:global Prop_Data
func PropData() PropType { return 0 }

// WeaponPipebombLauncher is TF_WEAPON_PIPEBOMBLAUNCHER.
//
//sp:global TF_WEAPON_PIPEBOMBLAUNCHER
func WeaponPipebombLauncher() Weapon { return 20 }

// AimCritical is CRITICAL, the aim priority a trap uses.
//
//sp:global CRITICAL
func AimCritical() int32 { return 2 }

// FeatureStickyStack is the switch between stacking the bombs and carpeting.
//
//sp:global FEATURE_STICKY_STACK
func FeatureStickyStack() int32 { return 3 }

// TravelDistanceToBombTarget is how far this ground is from where the bomb is
// going, along the route rather than through the walls.
//
//sp:plugin GetTravelDistanceToBombTarget
func TravelDistanceToBombTarget(area Area) float32 {
	if bombInfos.TravelDistanceToBombTarget == nil {
		missing("GetTravelDistanceToBombTarget")
	}
	return bombInfos.TravelDistanceToBombTarget(area)
}

// GetBombInfo fills the record, and says whether there was a bomb to fill it
// from.
//
//sp:plugin GetBombInfo
func GetBombInfo() (found bool, info BombInfo) {
	if bombInfos.GetBombInfo == nil {
		missing("GetBombInfo")
	}
	return bombInfos.GetBombInfo()
}

// BombHatchPosition is where the hatch is, which is the trap spot when no bomb
// is in play. Its SourcePawn returns the array.
//
//sp:body GetBombHatchPosition returns
func BombHatchPosition() [3]float32 {
	if bombInfos.BombHatchPosition == nil {
		missing("GetBombHatchPosition")
	}
	return bombInfos.BombHatchPosition()
}

// IsHeadAimingOnTarget says the bot is actually looking where it was told to.
//
//sp:method IsHeadAimingOnTarget
func (b Body) IsHeadAimingOnTarget() bool {
	if bombInfos.IsHeadAimingOnTarget == nil {
		missing("IBody.IsHeadAimingOnTarget")
	}
	return bombInfos.IsHeadAimingOnTarget(b)
}

// FeatureDemoStickySelfVeto is the switch on a demoman refusing the button when
// one of his own bombs is on top of him.
//
//sp:global FEATURE_DEMO_STICKY_SELF_VETO
func FeatureDemoStickySelfVeto() int32 { return 12 }

// IsBaseBoss says the entity really is a tank rather than something else the
// map called tank_boss.
//
//sp:plugin IsBaseBoss
func IsBaseBoss(entity int32) bool {
	if bombInfos.IsBaseBoss == nil {
		missing("IsBaseBoss")
	}
	return bombInfos.IsBaseBoss(entity)
}
