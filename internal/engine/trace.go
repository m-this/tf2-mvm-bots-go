package engine

/*
The trace, and the filter it runs.

A line of fire is a ray with a filter on it, and the filter is a function the game
calls back with an entity and the properties the caller set. SourceMod carries
those properties in a StringMap, so the map is a handle with a lifetime and the
filter is a function passed by name.
*/

// TraceCalls are the answers.
type TraceCalls struct {
	NewProperties          func() Properties
	SetProperty            func(p Properties, key string, value int32)
	PropertyValue          func(p Properties, key string) (bool, int32)
	CloseProperties        func(p Properties)
	TraceRayFilter         func(from [3]float32, to [3]float32, mask int32, rayType int32, filter func(entity int32, contentsMask int32, data Properties) bool, data Properties)
	TraceHull              func(from [3]float32, to [3]float32, mins [3]float32, maxs [3]float32, mask int32)
	DidHit                 func() bool
	TraceEntityIndex       func() int32
	StandardFilterRules    func(entity int32, contentsMask int32) bool
	PassServerEntityFilter func(entity int32, passEnt int32) bool
	ShouldCollide          func(entity int32, collisionGroup int32, contentsMask int32) bool
	GameRulesShouldCollide func(collisionGroup0 int32, collisionGroup1 int32) bool
	IsCombatItem           func(entity int32) bool
	IsCombatCharacter      func(e Entity) bool
	CollisionGroupOf       func(entity int32) int32
}

var traces TraceCalls

// InstallTraces puts a set of answers behind them.
func InstallTraces(c TraceCalls) func() {
	previous := traces
	Fill(&c)
	traces = c
	return func() { traces = previous }
}

// Properties is SourceMod's StringMap, which is how a trace filter is handed
// what the caller knows.
//
//sp:tag StringMap
type Properties int32

// MaskSolidBrushOnly is MASK_SOLID_BRUSHONLY: the world, and nothing standing in
// it.
//
//sp:global MASK_SOLID_BRUSHONLY
func MaskSolidBrushOnly() int32 { return 16395 }

// CollisionGroupNone is COLLISION_GROUP_NONE.
//
//sp:global COLLISION_GROUP_NONE
func CollisionGroupNone() int32 { return 0 }

// NewProperties makes the map a filter reads.
//
//sp:new StringMap
func NewProperties() Properties { return traces.NewProperties() }

// SetProperty writes one.
//
//sp:method SetValue
func (p Properties) SetProperty(key string, value int32) { traces.SetProperty(p, key, value) }

// Value reads one, and says whether it was there.
//
//sp:method GetValue
func (p Properties) Value(key string) (found bool, value int32) { return traces.PropertyValue(p, key) }

// Close releases the map.
//
//sp:method Close
func (p Properties) Close() { traces.CloseProperties(p) }

// TraceRayFilterData fires the ray with a filter that is handed the caller's
// properties, which is the six-argument form.
//
//sp:native TR_TraceRayFilter
//nolint:revive // unused-parameter: the filter is a name the emitter writes, not something the Go calls
func TraceRayFilterData(from [3]float32, to [3]float32, mask int32, rayType int32, filter func(entity int32, contentsMask int32, data Properties) bool, data Properties) {
	traces.TraceRayFilter(from, to, mask, rayType, filter, data)
}

// TraceHull sweeps a box from one point to the other, and leaves what it met
// for DidHit. The same box at both ends asks whether the box fits there.
//
//sp:native TR_TraceHull
func TraceHull(from [3]float32, to [3]float32, mins [3]float32, maxs [3]float32, mask int32) {
	traces.TraceHull(from, to, mins, maxs, mask)
}

// DidHit says the last ray hit something.
//
//sp:native TR_DidHit
func DidHit() bool { return traces.DidHit() }

// TraceEntityIndex is what it hit.
//
//sp:native TR_GetEntityIndex
func TraceEntityIndex() int32 { return traces.TraceEntityIndex() }

// StandardFilterRules is the game's own first pass over a candidate.
//
//sp:native StandardFilterRules
func StandardFilterRules(entity int32, contentsMask int32) bool {
	return traces.StandardFilterRules(entity, contentsMask)
}

// PassServerEntityFilter says the entity is not the one the ray came from.
//
//sp:native PassServerEntityFilter
func PassServerEntityFilter(entity int32, passEnt int32) bool {
	return traces.PassServerEntityFilter(entity, passEnt)
}

// ShouldCollide is the entity's own opinion about the collision group.
//
//sp:native ShouldCollide
func ShouldCollide(entity int32, collisionGroup int32, contentsMask int32) bool {
	return traces.ShouldCollide(entity, collisionGroup, contentsMask)
}

// GameRulesShouldCollide is the game rules' opinion about two groups.
//
//sp:library TFGameRules_ShouldCollide
func GameRulesShouldCollide(collisionGroup0 int32, collisionGroup1 int32) bool {
	return traces.GameRulesShouldCollide(collisionGroup0, collisionGroup1)
}

// IsCombatItem says the entity is a building or a projectile rather than scenery.
//
//sp:native BaseEntity_IsCombatItem
func IsCombatItem(entity int32) bool { return traces.IsCombatItem(entity) }

// IsCombatCharacter says the entity is somebody rather than something.
//
//sp:method IsCombatCharacter
func (e Entity) IsCombatCharacter() bool { return traces.IsCombatCharacter(e) }

// CollisionGroupOf is the group the entity is in.
//
//sp:native BaseEntity_GetCollisionGroup
func CollisionGroupOf(entity int32) int32 { return traces.CollisionGroupOf(entity) }
