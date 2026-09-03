package engine

// BluAssistCalls are the answers for bending a mission when few people turned up.
type BluAssistCalls struct {
	CreateAssistConVar func(name string, value string, description string, flags int32, hasMin bool, lowest float32, hasMax bool, highest float32) ConVar
	IsClientSourceTV   func(client int32) bool
	ClientUserID       func(client int32) int32
	ClientOfUserID     func(userid int32) int32
	ApplyNextFrame     func(callback func(userid int32), userid int32)
	RoundToCeil        func(value float32) int32
	SetEntityHealth    func(entity int32, health int32)
	SetEntPropData     func(entity int32, propType PropType, prop string, value int32)
	SetAttribByName    func(entity int32, name string, value float32)
}

var bluAssists BluAssistCalls

// InstallBluAssists puts a set of answers behind them.
func InstallBluAssists(c BluAssistCalls) func() {
	previous := bluAssists
	Fill(&c)
	bluAssists = c
	return func() { bluAssists = previous }
}

// FCVarNotify is FCVAR_NOTIFY, which prints the change to everybody.
//
//sp:global FCVAR_NOTIFY
func FCVarNotify() int32 { return 256 }

// CreateAssistConVar makes the one convar this lever is.
//
//sp:native CreateConVar
func CreateAssistConVar(name string, value string, description string, flags int32, hasMin bool, lowest float32, hasMax bool, highest float32) ConVar {
	return bluAssists.CreateAssistConVar(name, value, description, flags, hasMin, lowest, hasMax, highest)
}

// IsClientSourceTV says the slot is a recording, not a person.
//
//sp:native IsClientSourceTV
func IsClientSourceTV(client int32) bool { return bluAssists.IsClientSourceTV(client) }

// ClientUserID is the id that survives a client leaving, which is what a frame
// callback has to carry rather than the slot.
//
//sp:native GetClientUserId
func ClientUserID(client int32) int32 { return bluAssists.ClientUserID(client) }

// ClientOfUserID is the way back, and 0 when they have gone.
//
//sp:native GetClientOfUserId
func ClientOfUserID(userid int32) int32 { return bluAssists.ClientOfUserID(userid) }

// ApplyNextFrame runs the callback on the frame after this one, which is when
// the popfile has finished building the robot.
//
//sp:native RequestFrame
func ApplyNextFrame(callback func(userid int32), userid int32) {
	bluAssists.ApplyNextFrame(callback, userid)
}

// RoundToCeil rounds up, which is what a health scale wants: a robot worth
// nothing is not a robot.
//
//sp:native RoundToCeil
func RoundToCeil(value float32) int32 { return bluAssists.RoundToCeil(value) }

// SetEntityHealth writes the health it has now.
//
//sp:native SetEntityHealth
func SetEntityHealth(entity int32, health int32) { bluAssists.SetEntityHealth(entity, health) }

// SetEntPropData writes a datamap property.
//
//sp:native SetEntProp
func SetEntPropData(entity int32, propType PropType, prop string, value int32) {
	bluAssists.SetEntPropData(entity, propType, prop, value)
}

// SetAttribByName writes one, composing with what the mission already set.
//
//sp:native TF2Attrib_SetByName
func SetAttribByName(entity int32, name string, value float32) {
	bluAssists.SetAttribByName(entity, name, value)
}
