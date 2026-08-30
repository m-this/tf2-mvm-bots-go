package engine

// EntityCalls are the answers for the entity reads that have no better home.
type EntityCalls struct {
	DispatchUpdateTransmitState func(e Entity)
	AbsAngles                   func(e Entity) [3]float32
	FindDataMapInfo             func(entity int32, prop string) int32
	EntData                     func(entity int32, offset int32, size int32) int32
	EntPropAt                   func(entity int32, propType PropType, prop string, element int32) int32
	SetEntPropSend              func(entity int32, propType PropType, prop string, value int32)
}

var entities EntityCalls

// InstallEntities puts a set of answers behind them.
func InstallEntities(c EntityCalls) func() {
	previous := entities
	entities = c
	return func() { entities = previous }
}

// EffectNoDraw is EF_NODRAW, which is what hides a building being carried.
//
//sp:global EF_NODRAW
func EffectNoDraw() int32 { return 32 }

// ConditionRadiowave is TFCond_MVMBotRadiowave, the stun a bot takes from the
// radio wave.
//
//sp:global TFCond_MVMBotRadiowave
func ConditionRadiowave() Condition { return 90 }

// AbsAngles is which way the entity is facing.
//
//sp:method GetAbsAngles
func (e Entity) AbsAngles() (angles [3]float32) {
	if entities.AbsAngles == nil {
		missing("CBaseEntity.GetAbsAngles")
	}
	return entities.AbsAngles(e)
}

// DispatchUpdateTransmitState tells the game the entity's visibility changed.
//
//sp:method DispatchUpdateTransmitState
func (e Entity) DispatchUpdateTransmitState() {
	if entities.DispatchUpdateTransmitState == nil {
		missing("CBaseEntity.DispatchUpdateTransmitState")
	}
	entities.DispatchUpdateTransmitState(e)
}

// FindDataMapInfo is the offset of a datamap field, which is how a property with
// no send table is reached.
//
//sp:native FindDataMapInfo
func FindDataMapInfo(entity int32, prop string) int32 {
	if entities.FindDataMapInfo == nil {
		missing("FindDataMapInfo")
	}
	return entities.FindDataMapInfo(entity, prop)
}

// EntData reads that many bytes at an offset.
//
//sp:native GetEntData
func EntData(entity int32, offset int32, size int32) int32 {
	if entities.EntData == nil {
		missing("GetEntData")
	}
	return entities.EntData(entity, offset, size)
}

// EntPropAt is one element of an array property, which the wave stats are.
//
//sp:native GetEntProp after _
func EntPropAt(entity int32, propType PropType, prop string, element int32) int32 {
	if entities.EntPropAt == nil {
		missing("GetEntProp")
	}
	return entities.EntPropAt(entity, propType, prop, element)
}

// SetEntPropSend writes a networked property.
//
//sp:native SetEntProp
func SetEntPropSend(entity int32, propType PropType, prop string, value int32) {
	if entities.SetEntPropSend == nil {
		missing("SetEntProp")
	}
	entities.SetEntPropSend(entity, propType, prop, value)
}
