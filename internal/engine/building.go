package engine

/*
Building, and counting what the engineer has standing.

All of it is the plugin's own or stocklib's, and the disposable sentry is the
one thing in the mod that wants a different answer from GetObjectOfType, which
walks past disposable buildings on purpose.
*/

// BuildCalls are the answers.
type BuildCalls struct {
	PlayerObjectCountRaw       func(client int32) int32
	ModeOf                     func(building int32) ObjectMode
	IsPlasmaDisabled           func(building int32) bool
	DetonateObject             func(building int32)
	CreateEvent                func(name string) Event
	SetEventInt                func(e Event, key string, value int32)
	FireEvent                  func(e Event)
	ObjectOfType               func(client int32, objectType Object) int32
	IsBuilderSetTo             func(client int32, objectType Object) bool
	FakeClientCommandThrottled func(client int32, command string)
	IsPlacementOK              func(object int32) bool
	BuildStandPoint            func(spot [3]float32, from [3]float32, attempt int32, attempts int32, reach float32) (bool, [3]float32)
	PlayerObjectCount          func(client int32) int32
	PlayerObject               func(client int32, index int32) int32
	IsDisposableBuilding       func(object int32) bool
	AttribHookValueInt         func(value int32, name string, entity int32) int32
}

var builds BuildCalls

// InstallBuilds puts a set of answers behind them.
func InstallBuilds(c BuildCalls) func() {
	previous := builds
	Fill(&c)
	builds = c
	return func() { builds = previous }
}

// ObjectDispenser is TFObject_Dispenser.
//
//sp:global TFObject_Dispenser
func ObjectDispenser() Object { return 0 }

// ObjectOfType is the engineer's building of that kind, and
// INVALID_ENT_REFERENCE for none. It walks past disposable buildings.
//
//sp:body GetObjectOfType
func ObjectOfType(client int32, objectType Object) int32 {
	return builds.ObjectOfType(client, objectType)
}

// IsBuilderSetTo says the toolbox is already on that building.
//
//sp:body IsBuilderSetTo
func IsBuilderSetTo(client int32, objectType Object) bool {
	return builds.IsBuilderSetTo(client, objectType)
}

// FakeClientCommandThrottled sends the bot a console command, at a rate the
// server survives.
//
//sp:body FakeClientCommandThrottled
func FakeClientCommandThrottled(client int32, command string) {
	builds.FakeClientCommandThrottled(client, command)
}

// IsPlacementOK says the game would let the building go down there.
//
//sp:body IsPlacementOK
func IsPlacementOK(object int32) bool { return builds.IsPlacementOK(object) }

// BuildStandPoint walks a ring round a spot and answers where to stand for this
// attempt.
//
//sp:body BuildStandPoint
func BuildStandPoint(spot [3]float32, from [3]float32, attempt int32, attempts int32, reach float32) (ok bool, stand [3]float32) {
	return builds.BuildStandPoint(spot, from, attempt, attempts, reach)
}

// PlayerObjectCount is how many buildings the engineer has.
//
//sp:body PlayerObjectCount
func PlayerObjectCount(client int32) int32 { return builds.PlayerObjectCount(client) }

// PlayerObject is one of them.
//
//sp:native TF2Util_GetPlayerObject
func PlayerObject(client int32, index int32) int32 { return builds.PlayerObject(client, index) }

// IsDisposableBuilding says it is a mini rather than the real one.
//
//sp:native TF2_IsDisposableBuilding
func IsDisposableBuilding(object int32) bool { return builds.IsDisposableBuilding(object) }

// AttribHookValueInt is what an attribute makes of a value, which is how the
// mod asks whether an upgrade was bought.
//
//sp:native TF2Attrib_HookValueInt
func AttribHookValueInt(value int32, name string, entity int32) int32 {
	return builds.AttribHookValueInt(value, name, entity)
}

// PlayerObjectCountRaw is the game's own count, which throws for a client who
// has left: PlayerObjectCount is the wrapper that answers zero instead.
//
//sp:native TF2Util_GetPlayerObjectCount
func PlayerObjectCountRaw(client int32) int32 { return builds.PlayerObjectCountRaw(client) }

// ModeOf is which half of a teleporter a building is.
//
//sp:library TF2_GetObjectMode
func ModeOf(building int32) ObjectMode { return builds.ModeOf(building) }

// IsPlasmaDisabled says a Short Circuit has knocked the building out.
//
//sp:library TF2_IsPlasmaDisabled
func IsPlasmaDisabled(building int32) bool { return builds.IsPlasmaDisabled(building) }

// DetonateObject takes the building down.
//
//sp:library TF2_DetonateObject
func DetonateObject(building int32) { builds.DetonateObject(building) }

// Event is a game event this port fires, which the statistics plugin and the
// game itself both listen for.
//
//sp:tag Event
type Event int32

// NoEvent is null, which is what CreateEvent answers when the game does not know
// the name.
//
//sp:global null
func NoEvent() Event { return 0 }

// CreateEvent makes one.
//
//sp:native CreateEvent
func CreateEvent(name string) Event { return builds.CreateEvent(name) }

// SetEventInt writes one of its fields.
//
//sp:method SetInt
func (e Event) SetEventInt(key string, value int32) { builds.SetEventInt(e, key, value) }

// Fire sends it, and the handle goes with it.
//
//sp:method Fire
func (e Event) Fire() { builds.FireEvent(e) }
