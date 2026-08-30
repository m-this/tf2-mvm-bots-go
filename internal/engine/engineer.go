package engine

/*
The engineer's own seam: the buildings he carries, the busters he carries them
away from, and the four behaviours his idle action suspends into.

The four constructors are //sp:body rather than //sp:plugin. Each of them is
generated from its own package now, so naming one here is naming a function this
port owns, and the generator refuses the same name being owned twice.
*/

// EngineerCalls are the answers.
type EngineerCalls struct {
	PickBusterRetreatArea  func(sentry int32, buster int32) Area
	DetonateObjectOfType   func(client int32, objectType Object)
	CarriedObject          func(client int32) int32
	IsMiniBuilding         func(building int32) bool
	IsBuildingUp           func(building int32) bool
	UpgradeLevel           func(building int32) int32
	EntityHealth           func(entity int32) int32
	EntityMaxHealth        func(entity int32) int32
	IsRescueRangerEquipped func(client int32) bool
	TurretAngles           func(sentry int32) [3]float32
	ShouldRelocateNest     func(client int32) (bool, Area)
	ShouldBuildDisposable  func(actor int32) bool
	ShouldBuildTeleporter  func(actor int32) bool
	AbsAnglesOf            func(entity int32) [3]float32
	NestRelocateOf         func(client int32) Area
	EngineerGunSpendsMetal func(client int32) bool
	SetNestRelocate        func(client int32, area Area)
	RunScriptCode          func(client int32, code string)
	HeadSteadyDuration     func(b Body) float32
	SetPathGoalEntity      func(bot PluginBot, entity int32)
	EntIndexToEntRef       func(entity int32) int32
	EntRefToEntIndex       func(ref int32) int32
	AmmoOf                 func(client int32, propType PropType, prop string, element int32) int32
	BuildSentrygun         func() Behaviour
	BuildDispenser         func() Behaviour
	BuildTeleporter        func() Behaviour
	BuildDisposable        func() Behaviour
	CreateRepeatingTimer   func(interval float32) Timer
	KillTimer              func(timer Timer)
}

var engineers EngineerCalls

// InstallEngineers puts a set of answers behind them.
func InstallEngineers(c EngineerCalls) func() {
	previous := engineers
	engineers = c
	return func() { engineers = previous }
}

/*
Timer is SourcePawn's Handle for a repeating timer.

It is not one of this port's handles: a timer is ended by KillTimer, not by
delete, and the one here is held in a global rather than in a local, so nothing
about the defer rule applies to it.

//sp:tag Handle
*/
type Timer int32

// NoTimer is null, which is what the global holds when nothing is queued.
//
//sp:global null
func NoTimer() Timer { return 0 }

// BusterHaulRange is how far away a buster still is when carrying the sentry
// off is worth starting.
//
//sp:global BUSTER_HAUL_RANGE
func BusterHaulRange() float32 { return 1800.0 }

// NestRelocate is redbots_manager_engineer_nest_relocate, the switch on asking
// whether the ground an engineer holds is still the ground he wants.
//
//sp:global redbots_manager_engineer_nest_relocate
func NestRelocate() ConVar { return 0 }

/*
Default is SourcePawn's underscore, which is how a call site says "the
declaration's own default" for an argument it does not care about.

Go has no such thing, so it is spelled as a value: the one place the plugin
writes it in the middle of an argument list rather than at the end of one, which
is the script-code call the wrangler uses.

//sp:global _
*/
func Default() int32 { return 0 }

// TimerRepeat is TIMER_REPEAT.
//
//sp:global TIMER_REPEAT
func TimerRepeat() int32 { return 1 }

// PluginStop is Plugin_Stop, which is what a repeating timer returns to end.
//
//sp:global Plugin_Stop
func PluginStop() Outcome { return 4 }

// PluginContinue is Plugin_Continue.
//
//sp:global Plugin_Continue
func PluginContinue() Outcome { return 0 }

/*
CreateNestRelocateTimer starts the one timer this port owns.

SourcePawn takes the callback as an argument and the subset has no function
value, so the callback and the repeat flag are written into the directive: this
extern is that one call and nothing else, and the function it names is generated
beside it.

//sp:native CreateTimer after Timer_EvaluateNestRelocation _ TIMER_REPEAT
*/
func CreateNestRelocateTimer(interval float32) Timer {
	if engineers.CreateRepeatingTimer == nil {
		missing("CreateTimer")
	}
	return engineers.CreateRepeatingTimer(interval)
}

// KillTimer stops it.
//
//sp:native KillTimer
func KillTimer(timer Timer) {
	if engineers.KillTimer == nil {
		missing("KillTimer")
	}
	engineers.KillTimer(timer)
}

// AbsAnglesOf is which way the entity is facing.
//
//sp:plugin GetAbsAngles returns
func AbsAnglesOf(entity int32) (angles [3]float32) {
	if engineers.AbsAnglesOf == nil {
		missing("GetAbsAngles")
	}
	return engineers.AbsAnglesOf(entity)
}

// ShouldBuildTeleporter is the teleporter behaviour's own precondition.
//
//sp:body ShouldBuildTeleporter
func ShouldBuildTeleporter(actor int32) bool {
	if engineers.ShouldBuildTeleporter == nil {
		missing("ShouldBuildTeleporter")
	}
	return engineers.ShouldBuildTeleporter(actor)
}

// EngineerGunSpendsMetal is an engineer carrying a gun that costs metal to
// fire: the Widowmaker, the Short Circuit and the Rescue Ranger.
//
//sp:plugin EngineerGunSpendsMetal
func EngineerGunSpendsMetal(client int32) bool {
	if engineers.EngineerGunSpendsMetal == nil {
		missing("EngineerGunSpendsMetal")
	}
	return engineers.EngineerGunSpendsMetal(client)
}

// NestRelocateOf is the ground the between-waves answer wants this engineer to
// move to, NULL_AREA for none. The array is util.sp's, not the idle action's.
//
//sp:slot m_aNestAreaRelocate
func NestRelocateOf(client int32) Area {
	if engineers.NestRelocateOf == nil {
		missing("m_aNestAreaRelocate")
	}
	return engineers.NestRelocateOf(client)
}

// SetNestRelocate writes it.
//
//sp:slotset m_aNestAreaRelocate
func SetNestRelocate(client int32, area Area) {
	if engineers.SetNestRelocate == nil {
		missing("m_aNestAreaRelocate")
	}
	engineers.SetNestRelocate(client, area)
}

// PickBusterRetreatArea is ground out of the blast to carry the sentry to.
//
//sp:plugin PickBusterRetreatArea
func PickBusterRetreatArea(sentry int32, buster int32) Area {
	if engineers.PickBusterRetreatArea == nil {
		missing("PickBusterRetreatArea")
	}
	return engineers.PickBusterRetreatArea(sentry, buster)
}

// DetonateObjectOfType takes one of the engineer's buildings down.
//
//sp:plugin DetonateObjectOfType
func DetonateObjectOfType(client int32, objectType Object) {
	if engineers.DetonateObjectOfType == nil {
		missing("DetonateObjectOfType")
	}
	engineers.DetonateObjectOfType(client, objectType)
}

// CarriedObject is what is in his hands, and -1 for nothing.
//
//sp:plugin TF2_GetCarriedObject
func CarriedObject(client int32) int32 {
	if engineers.CarriedObject == nil {
		missing("TF2_GetCarriedObject")
	}
	return engineers.CarriedObject(client)
}

// IsMiniBuilding says it came out of a Gunslinger, so it is rebuilt rather than
// nursed.
//
//sp:plugin TF2_IsMiniBuilding
func IsMiniBuilding(building int32) bool {
	if engineers.IsMiniBuilding == nil {
		missing("TF2_IsMiniBuilding")
	}
	return engineers.IsMiniBuilding(building)
}

// IsBuildingUp says the thing is still going up.
//
//sp:plugin TF2_IsBuilding
func IsBuildingUp(building int32) bool {
	if engineers.IsBuildingUp == nil {
		missing("TF2_IsBuilding")
	}
	return engineers.IsBuildingUp(building)
}

// UpgradeLevel is one, two or three.
//
//sp:plugin TF2_GetUpgradeLevel
func UpgradeLevel(building int32) int32 {
	if engineers.UpgradeLevel == nil {
		missing("TF2_GetUpgradeLevel")
	}
	return engineers.UpgradeLevel(building)
}

// EntityHealth is what it has left.
//
//sp:plugin BaseEntity_GetHealth
func EntityHealth(entity int32) int32 {
	if engineers.EntityHealth == nil {
		missing("BaseEntity_GetHealth")
	}
	return engineers.EntityHealth(entity)
}

// EntityMaxHealth is what it had.
//
//sp:native TF2Util_GetEntityMaxHealth
func EntityMaxHealth(entity int32) int32 {
	if engineers.EntityMaxHealth == nil {
		missing("TF2Util_GetEntityMaxHealth")
	}
	return engineers.EntityMaxHealth(entity)
}

// IsRescueRangerEquipped says he can repair from behind cover.
//
//sp:plugin TF2_IsRescueRangerEquipped
func IsRescueRangerEquipped(client int32) bool {
	if engineers.IsRescueRangerEquipped == nil {
		missing("TF2_IsRescueRangerEquipped")
	}
	return engineers.IsRescueRangerEquipped(client)
}

// TurretAngles is where the sentry is pointing, which is what the engineer
// stands behind.
//
//sp:plugin GetTurretAngles
func TurretAngles(sentry int32) (angles [3]float32) {
	if engineers.TurretAngles == nil {
		missing("GetTurretAngles")
	}
	return engineers.TurretAngles(sentry)
}

// ShouldRelocateNest is the between-waves question, and the ground it answers
// with.
//
//sp:plugin ShouldRelocateNest
func ShouldRelocateNest(client int32) (yes bool, destination Area) {
	if engineers.ShouldRelocateNest == nil {
		missing("ShouldRelocateNest")
	}
	return engineers.ShouldRelocateNest(client)
}

// ShouldBuildDisposable says a second gun beside the first is worth the metal.
//
//sp:body ShouldBuildDisposable
func ShouldBuildDisposable(actor int32) bool {
	if engineers.ShouldBuildDisposable == nil {
		missing("ShouldBuildDisposable")
	}
	return engineers.ShouldBuildDisposable(actor)
}

// RunScriptCode hands a line of VScript to the bot, which is the only way to
// press two buttons on the same frame while the sentry is wrangled.
//
//sp:plugin OSLib_RunScriptCode
//nolint:revive // unused-parameter: the two are SourcePawn's own defaults, written through
func RunScriptCode(client int32, first int32, second int32, code string) {
	if engineers.RunScriptCode == nil {
		missing("OSLib_RunScriptCode")
	}
	engineers.RunScriptCode(client, code)
}

// HeadSteadyDuration is how long the head has been pointing where it points.
//
//sp:method GetHeadSteadyDuration
func (b Body) HeadSteadyDuration() float32 {
	if engineers.HeadSteadyDuration == nil {
		missing("IBody.GetHeadSteadyDuration")
	}
	return engineers.HeadSteadyDuration(b)
}

// SetPathGoalEntity walks the bot at a thing rather than at a place.
//
//sp:method SetPathGoalEntity
func (p PluginBot) SetPathGoalEntity(entity int32) {
	if engineers.SetPathGoalEntity == nil {
		missing("PluginBot.SetPathGoalEntity")
	}
	engineers.SetPathGoalEntity(p, entity)
}

// EntIndexToEntRef is the reference that survives the entity being deleted.
//
//sp:native EntIndexToEntRef
func EntIndexToEntRef(entity int32) int32 {
	if engineers.EntIndexToEntRef == nil {
		missing("EntIndexToEntRef")
	}
	return engineers.EntIndexToEntRef(entity)
}

// EntRefToEntIndex is the way back, and INVALID_ENT_REFERENCE when the thing is
// gone.
//
//sp:native EntRefToEntIndex
func EntRefToEntIndex(ref int32) int32 {
	if engineers.EntRefToEntIndex == nil {
		missing("EntRefToEntIndex")
	}
	return engineers.EntRefToEntIndex(ref)
}

// AmmoOf is one slot of the player's ammo array, which needs the element index
// a plain property read does not take.
//
//sp:native GetEntProp after 4
func AmmoOf(client int32, propType PropType, prop string, element int32) int32 {
	if engineers.AmmoOf == nil {
		missing("GetEntProp")
	}
	return engineers.AmmoOf(client, propType, prop, element)
}

// BuildSentrygun is the behaviour that stands one up.
//
//sp:body CTFBotMvMEngineerBuildSentrygun
func BuildSentrygun() Behaviour {
	if engineers.BuildSentrygun == nil {
		missing("CTFBotMvMEngineerBuildSentrygun")
	}
	return engineers.BuildSentrygun()
}

// BuildDispenser is the one that feeds it.
//
//sp:body CTFBotMvMEngineerBuildDispenser
func BuildDispenser() Behaviour {
	if engineers.BuildDispenser == nil {
		missing("CTFBotMvMEngineerBuildDispenser")
	}
	return engineers.BuildDispenser()
}

// BuildTeleporter is what he does with the time left over.
//
//sp:body CTFBotMvMEngineerBuildTeleporter
func BuildTeleporter() Behaviour {
	if engineers.BuildTeleporter == nil {
		missing("CTFBotMvMEngineerBuildTeleporter")
	}
	return engineers.BuildTeleporter()
}

// BuildDisposable stands a mini beside the nest.
//
//sp:body CTFBotMvMEngineerBuildDisposable
func BuildDisposable() Behaviour {
	if engineers.BuildDisposable == nil {
		missing("CTFBotMvMEngineerBuildDisposable")
	}
	return engineers.BuildDisposable()
}
