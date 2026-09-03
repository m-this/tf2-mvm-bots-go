package engine

// FaultCalls are the answers for the injected faults.
type FaultCalls struct {
	CreatePlainConVar func(name string, value string, description string, flags int32) ConVar
	RegServerCmd      func(name string, callback func(args int32) Outcome)
	ConVarString      func(c ConVar) Text
	StrEqualFold      func(a Text, b Text, caseSensitive bool) bool
	RawClassName      func(class Class) Text
	TeleportEntity    func(entity int32, origin [3]float32, angles [3]float32, velocity [3]float32)
	HasSniperRifle    func(client int32) bool
	SniperSpots       func() List
}

var faults FaultCalls

// InstallFaults puts a set of answers behind them.
func InstallFaults(c FaultCalls) func() {
	previous := faults
	Fill(&c)
	faults = c
	return func() { faults = previous }
}

// CreatePlainConVar makes one with no bounds, which is what a text setting takes.
//
//sp:native CreateConVar
func CreatePlainConVar(name string, value string, description string, flags int32) ConVar {
	return faults.CreatePlainConVar(name, value, description, flags)
}

// RegServerCmd registers a console command, taking its callback by name.
//
//sp:native RegServerCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegServerCmd(name string, callback func(args int32) Outcome) {
	faults.RegServerCmd(name, callback)
}

// StringValue is a convar's value as text, filled into a buffer the caller
// declares.
//
//sp:method GetString sized
func (c ConVar) StringValue() (out Text) { return faults.ConVarString(c) }

// StrEqualFold compares two buffers, case sensitively or not.
//
//sp:native StrEqual
func StrEqualFold(a Text, b Text, caseSensitive bool) bool {
	return faults.StrEqualFold(a, b, caseSensitive)
}

// RawClassName is the game's own name for a class, which is what a convar
// naming one is compared against.
//
//sp:slot g_sRawPlayerClassNames
func RawClassName(class Class) Text { return faults.RawClassName(class) }

// TeleportEntity puts something where it was, which is how a wedge is held.
//
//sp:native TeleportEntity
func TeleportEntity(entity int32, origin [3]float32, angles [3]float32, velocity [3]float32) {
	faults.TeleportEntity(entity, origin, angles, velocity)
}

// HasSniperRifle says the bot is carrying one.
//
//sp:body HasSniperRifle
func HasSniperRifle(client int32) bool { return faults.HasSniperRifle(client) }

// SniperSpots are the sniper positions the map configuration names.
//
//sp:global g_arrMapConfig.adtSniperSpot
func SniperSpots() List { return 0 }
