package engine

// PlayerCalls are the answers about players and the server they are on.
type PlayerCalls struct {
	SquareRoot          func(value float32) float32
	VectorLengthSquared func(v [3]float32, squared bool) float32
	EyePositionOf       func(client int32) [3]float32
	FakeClientCommand   func(client int32, format string, args ...any)
	FindConVar          func(name string) ConVar
	NewKeyValues        func(name string) KeyValues
	FakeCommandKV       func(client int32, kv KeyValues)
	CloseKeyValues      func(kv KeyValues)
	ClientCount         func(inGameOnly bool) int32
	ResourceEntity      func() int32
	SetVariantString    func(value string)
	AcceptEntityInput   func(entity int32, input string, activator int32, caller int32, outputID int32) bool
	TextLength          func(text Text) int32
	TextLengthOf        func(text string) int32
}

var players PlayerCalls

// InstallPlayers puts a set of answers behind them.
func InstallPlayers(c PlayerCalls) func() {
	previous := players
	players = c
	return func() { players = previous }
}

// KeyValues is SourceMod's KeyValues, which a fake command is built from.
//
//sp:tag KeyValues
type KeyValues int32

// SquareRoot is the square root.
//
//sp:native SquareRoot
func SquareRoot(value float32) float32 {
	if players.SquareRoot == nil {
		missing("SquareRoot")
	}
	return players.SquareRoot(value)
}

// VectorLengthSquared is the length, or its square when asked: the square is the
// cheap one and a comparison rarely needs the root.
//
//sp:native GetVectorLength
func VectorLengthSquared(v [3]float32, squared bool) float32 {
	if players.VectorLengthSquared == nil {
		missing("GetVectorLength")
	}
	return players.VectorLengthSquared(v, squared)
}

// EyePositionOf is where the client is looking from, filled into a buffer.
//
//sp:plugin BaseEntity_EyePosition
func EyePositionOf(client int32) (position [3]float32) {
	if players.EyePositionOf == nil {
		missing("BaseEntity_EyePosition")
	}
	return players.EyePositionOf(client)
}

// FakeClientCommand issues a console command as the client, unthrottled.
//
//sp:native FakeClientCommand
func FakeClientCommand(client int32, format string, args ...any) {
	if players.FakeClientCommand == nil {
		missing("FakeClientCommand")
	}
	players.FakeClientCommand(client, format, args...)
}

// FindConVar is a convar by name, and null when nothing declares it, which is how
// another plugin is detected.
//
//sp:native FindConVar
func FindConVar(name string) ConVar {
	if players.FindConVar == nil {
		missing("FindConVar")
	}
	return players.FindConVar(name)
}

// NewKeyValues makes one.
//
//sp:new KeyValues
func NewKeyValues(name string) KeyValues {
	if players.NewKeyValues == nil {
		missing("new KeyValues")
	}
	return players.NewKeyValues(name)
}

// Close releases it.
//
//sp:delete Close
func (kv KeyValues) Close() {
	if players.CloseKeyValues == nil {
		missing("delete KeyValues")
	}
	players.CloseKeyValues(kv)
}

// FakeCommandKV sends a command the game reads as key values, which is how the
// action slot is used.
//
//sp:native FakeClientCommandKeyValues
func FakeCommandKV(client int32, kv KeyValues) {
	if players.FakeCommandKV == nil {
		missing("FakeClientCommandKeyValues")
	}
	players.FakeCommandKV(client, kv)
}

// ClientCount is how many are connected.
//
//sp:native GetClientCount
func ClientCount(inGameOnly bool) int32 {
	if players.ClientCount == nil {
		missing("GetClientCount")
	}
	return players.ClientCount(inGameOnly)
}

// ResourceEntity is the player resource, which carries the per-client tables.
//
//sp:native GetPlayerResourceEntity
func ResourceEntity() int32 {
	if players.ResourceEntity == nil {
		missing("GetPlayerResourceEntity")
	}
	return players.ResourceEntity()
}

// PlayerSideSpeed is PLAYER_SIDESPEED, the strafe speed a bot is pushed at.
//
//sp:global PLAYER_SIDESPEED
func PlayerSideSpeed() float32 { return 450.0 }

// SetVariantString sets the value the next entity input carries.
//
//sp:native SetVariantString
func SetVariantString(value string) {
	if players.SetVariantString == nil {
		missing("SetVariantString")
	}
	players.SetVariantString(value)
}

// AcceptEntityInput fires an input on an entity.
//
//sp:native AcceptEntityInput
func AcceptEntityInput(entity int32, input string, activator int32, caller int32, outputID int32) bool {
	if players.AcceptEntityInput == nil {
		missing("AcceptEntityInput")
	}
	return players.AcceptEntityInput(entity, input, activator, caller, outputID)
}

// TextLengthOf is how long a literal is, which is how an empty default is told
// from a real attachment point.
//
//sp:native strlen
func TextLengthOf(text string) int32 {
	if players.TextLengthOf == nil {
		missing("strlen")
	}
	return players.TextLengthOf(text)
}

// TextLength is how long a buffer's text is.
//
//sp:native strlen
func TextLength(text Text) int32 {
	if players.TextLength == nil {
		missing("strlen")
	}
	return players.TextLength(text)
}
