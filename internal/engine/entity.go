package engine

// EntityCalls are the answers for the entity reads that have no better home.
type EntityCalls struct {
	DispatchUpdateTransmitState func(e Entity)
	AbsAngles                   func(e Entity) [3]float32
	ClientName                  func(client int32) (bool, Text)
	MakeVectorFromPoints        func(from [3]float32, to [3]float32) [3]float32
	VectorAngles                func(direction [3]float32) [3]float32
	RoundToFloor                func(value float32) int32
	GameRulesPropAt             func(prop string, size int32, element int32) int32
	CreateEntityByName          func(classname string) int32
	RemoveEffectsFrom           func(entity int32, effects int32)
	DispatchSpawn               func(entity int32) bool
	SetEntPropEnt               func(entity int32, propType PropType, prop string, other int32)
	SetObjectMode               func(building int32, mode int32)
	TeamMayCapturePoint         func(team Team, pointIndex int32) bool
	CompareText                 func(a Text, b Text, caseSensitive bool) int32
	LoadFromAddress             func(address Address) int32
	FindSendPropInfo            func(class string, prop string) int32
	SetEntData                  func(entity int32, offset int32, value int32)
	SetEntPropAt                func(entity int32, propType PropType, prop string, value int32, element int32)
	IsEntityWearable            func(entity int32) bool
	EquipPlayerWearable         func(client int32, item int32)
	EquipPlayerWeapon           func(client int32, item int32)
	SetItemID                   func(item int32, id int32)
	IsItemDefIndexSapper        func(itemDefIndex int32) bool
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

// ClientName is what the player calls themselves, and whether the game answered
// at all: the native fills the buffer and returns a bool, and the plugin reads
// both.
//
//sp:native GetClientName sized
func ClientName(client int32) (ok bool, name Text) {
	if entities.ClientName == nil {
		missing("GetClientName")
	}
	return entities.ClientName(client)
}

// MakeVectorFromPoints is the vector from one point to another.
//
//sp:native MakeVectorFromPoints
func MakeVectorFromPoints(from [3]float32, to [3]float32) (direction [3]float32) {
	if entities.MakeVectorFromPoints == nil {
		missing("MakeVectorFromPoints")
	}
	return entities.MakeVectorFromPoints(from, to)
}

// VectorAngles is the angles a direction points at.
//
//sp:native GetVectorAngles
func VectorAngles(direction [3]float32) (angles [3]float32) {
	if entities.VectorAngles == nil {
		missing("GetVectorAngles")
	}
	return entities.VectorAngles(direction)
}

// RoundToFloor rounds down.
//
//sp:native RoundToFloor
func RoundToFloor(value float32) int32 {
	if entities.RoundToFloor == nil {
		missing("RoundToFloor")
	}
	return entities.RoundToFloor(value)
}

// GameRulesPropAt is one element of a game rules array property.
//
//sp:native GameRules_GetProp
func GameRulesPropAt(prop string, size int32, element int32) int32 {
	if entities.GameRulesPropAt == nil {
		missing("GameRules_GetProp")
	}
	return entities.GameRulesPropAt(prop, size, element)
}

// Address is a raw memory address, which the gamedata hands out and two reads
// follow.
//
//sp:tag Address
type Address int32

// RemoveEffectsFrom clears effect bits, which internal/body/entity generates.
//
//sp:body RemoveEffects
func RemoveEffectsFrom(entity int32, effects int32) {
	if entities.RemoveEffectsFrom == nil {
		missing("RemoveEffects")
	}
	entities.RemoveEffectsFrom(entity, effects)
}

// CreateEntityByName makes one, unspawned.
//
//sp:native CreateEntityByName
func CreateEntityByName(classname string) int32 {
	if entities.CreateEntityByName == nil {
		missing("CreateEntityByName")
	}
	return entities.CreateEntityByName(classname)
}

// DispatchSpawn brings it into the world.
//
//sp:native DispatchSpawn
func DispatchSpawn(entity int32) bool {
	if entities.DispatchSpawn == nil {
		missing("DispatchSpawn")
	}
	return entities.DispatchSpawn(entity)
}

// SetEntPropEnt points one entity property at another.
//
//sp:native SetEntPropEnt
func SetEntPropEnt(entity int32, propType PropType, prop string, other int32) {
	if entities.SetEntPropEnt == nil {
		missing("SetEntPropEnt")
	}
	entities.SetEntPropEnt(entity, propType, prop, other)
}

// SetObjectMode says which half of a teleporter a building is, which a sapper
// copies from what it is sapping.
//
//sp:plugin TF2_SetObjectMode
func SetObjectMode(building int32, mode int32) {
	if entities.SetObjectMode == nil {
		missing("TF2_SetObjectMode")
	}
	entities.SetObjectMode(building, mode)
}

// TeamMayCapturePoint says the point is one this team is allowed to take.
//
//sp:plugin TFGameRules_TeamMayCapturePoint
func TeamMayCapturePoint(team Team, pointIndex int32) bool {
	if entities.TeamMayCapturePoint == nil {
		missing("TFGameRules_TeamMayCapturePoint")
	}
	return entities.TeamMayCapturePoint(team, pointIndex)
}

// CompareText is strcmp, which answers zero when two buffers hold the same text.
//
//sp:native strcmp
func CompareText(a Text, b Text, caseSensitive bool) int32 {
	if entities.CompareText == nil {
		missing("strcmp")
	}
	return entities.CompareText(a, b, caseSensitive)
}

// LoadFromAddress reads memory, which is how an offset the gamedata names is
// followed.
//
//sp:native LoadFromAddress after NumberType_Int32
func LoadFromAddress(address Address) int32 {
	if entities.LoadFromAddress == nil {
		missing("LoadFromAddress")
	}
	return entities.LoadFromAddress(address)
}

// FindSendPropInfo is the offset of a networked property, which is how one
// SetEntProp refuses to write is reached.
//
//sp:native FindSendPropInfo
func FindSendPropInfo(class string, prop string) int32 {
	if entities.FindSendPropInfo == nil {
		missing("FindSendPropInfo")
	}
	return entities.FindSendPropInfo(class, prop)
}

// SetEntData writes at an offset.
//
//sp:native SetEntData
func SetEntData(entity int32, offset int32, value int32) {
	if entities.SetEntData == nil {
		missing("SetEntData")
	}
	entities.SetEntData(entity, offset, value)
}

// SetEntPropAt writes one element of an array property.
//
//sp:native SetEntProp after _
func SetEntPropAt(entity int32, propType PropType, prop string, value int32, element int32) {
	if entities.SetEntPropAt == nil {
		missing("SetEntProp")
	}
	entities.SetEntPropAt(entity, propType, prop, value, element)
}

// IsEntityWearable says the item is worn rather than held.
//
//sp:native TF2Util_IsEntityWearable
func IsEntityWearable(entity int32) bool {
	if entities.IsEntityWearable == nil {
		missing("TF2Util_IsEntityWearable")
	}
	return entities.IsEntityWearable(entity)
}

// EquipPlayerWearable puts a worn item on.
//
//sp:native TF2Util_EquipPlayerWearable
func EquipPlayerWearable(client int32, item int32) {
	if entities.EquipPlayerWearable == nil {
		missing("TF2Util_EquipPlayerWearable")
	}
	entities.EquipPlayerWearable(client, item)
}

// EquipPlayerWeapon puts a held item in the hands.
//
//sp:native EquipPlayerWeapon
func EquipPlayerWeapon(client int32, item int32) {
	if entities.EquipPlayerWeapon == nil {
		missing("EquipPlayerWeapon")
	}
	entities.EquipPlayerWeapon(client, item)
}

// SetItemID stamps an item with an id, which the game wants unique enough not to
// collide with a real one.
//
//sp:plugin EconItemView_SetItemID
func SetItemID(item int32, id int32) {
	if entities.SetItemID == nil {
		missing("EconItemView_SetItemID")
	}
	entities.SetItemID(item, id)
}

// IsItemDefIndexSapper says the item is a sapper, which internal/body/reflect
// generates.
//
//sp:body IsItemDefIndexSapper
func IsItemDefIndexSapper(itemDefIndex int32) bool {
	if entities.IsItemDefIndexSapper == nil {
		missing("IsItemDefIndexSapper")
	}
	return entities.IsItemDefIndexSapper(itemDefIndex)
}
