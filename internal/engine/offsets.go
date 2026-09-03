package engine

/*
Reading a field the game has and SourceMod does not name.

An offset comes from the gamedata file, sometimes as a number and sometimes as
a number past a send prop the game does name. Either way it is looked up once at
load and read out of a map afterwards, because FindSendPropInfo is not something
to do per frame.
*/

// OffsetCalls are the answers.
type OffsetCalls struct {
	NewProperties        func() Properties
	EntDataDefault       func(entity int32, offset int32) int32
	EntDataSized         func(entity int32, offset int32, size int32) int32
	SetEntDataSized      func(entity int32, offset int32, value Cell, size int32)
	EntDataVector        func(entity int32, offset int32) [3]float32
	EntDataFloat         func(entity int32, offset int32) float32
	EntityAddress        func(entity int32) Address
	LoadFromAddress      func(address Address, size int32) int32
	GameDataOffset       func(g GameData, key string) int32
	GameDataOffsetText   func(g GameData, key Text) int32
	GameDataKeyValue     func(g GameData, key Text) (bool, Text)
	FormatInto           func(format string, args []any) Text
	SetPropertyAt        func(p Properties, key Text, value int32)
	ValueAt              func(p Properties, key Text) (bool, int32)
	OffsetsMap           func() Properties
	SetOffsetMap         func(p Properties)
	OffsetMap            func() Properties
	FindSendPropInfoText func(class string, prop Text) int32
	StrEqualLiterals     func(a string, b string) bool
	ThrowErrorText       func(format string, args []any)
	LoadFloatFromAddress func(address Address) float32
}

var offsets OffsetCalls

// InstallOffsets puts a set of answers behind them.
func InstallOffsets(c OffsetCalls) func() {
	previous := offsets
	offsets = c
	return func() { offsets = previous }
}

// EntDataDefault reads a cell at an offset, which is what GetEntData does when
// nobody says a size.
//
//sp:native GetEntData
func EntDataDefault(entity int32, offset int32) int32 {
	if offsets.EntDataDefault == nil {
		missing("GetEntData")
	}
	return offsets.EntDataDefault(entity, offset)
}

// EntDataSized reads that many bytes at an offset.
//
//sp:native GetEntData
func EntDataSized(entity int32, offset int32, size int32) int32 {
	if offsets.EntDataSized == nil {
		missing("GetEntData")
	}
	return offsets.EntDataSized(entity, offset, size)
}

// SetEntDataSized writes that many bytes at an offset.
//
//sp:native SetEntData
func SetEntDataSized(entity int32, offset int32, value Cell, size int32) {
	if offsets.SetEntDataSized == nil {
		missing("SetEntData")
	}
	offsets.SetEntDataSized(entity, offset, value, size)
}

// EntDataVector reads three floats at an offset.
//
//sp:native GetEntDataVector
func EntDataVector(entity int32, offset int32) (out [3]float32) {
	if offsets.EntDataVector == nil {
		missing("GetEntDataVector")
	}
	return offsets.EntDataVector(entity, offset)
}

// EntDataFloat reads a float at an offset.
//
//sp:native GetEntDataFloat
func EntDataFloat(entity int32, offset int32) float32 {
	if offsets.EntDataFloat == nil {
		missing("GetEntDataFloat")
	}
	return offsets.EntDataFloat(entity, offset)
}

// EntityAddress is where the entity lives, which is what an offset is added to
// when the field is not one SourceMod can reach by index.
//
//sp:native GetEntityAddress
func EntityAddress(entity int32) Address {
	if offsets.EntityAddress == nil {
		missing("GetEntityAddress")
	}
	return offsets.EntityAddress(entity)
}

// NumberTypeInt32 is NumberType_Int32, four bytes.
//
//sp:global NumberType_Int32
func NumberTypeInt32() int32 { return 0 }

// Offset is the number the gamedata file gives for that key.
//
//sp:method GetOffset
func (g GameData) Offset(key Text) int32 {
	if offsets.GameDataOffsetText == nil {
		missing("GameData.GetOffset")
	}
	return offsets.GameDataOffsetText(g, key)
}

// KeyValue is the text the gamedata file gives for that key, and whether it
// gives one at all.
//
//sp:method GetKeyValue sized
func (g GameData) KeyValue(key Text) (found bool, value Text) {
	if offsets.GameDataKeyValue == nil {
		missing("GameData.GetKeyValue")
	}
	return offsets.GameDataKeyValue(g, key)
}

// FormatInto writes a formatted line into a buffer that already exists.
//
//sp:native Format fills
func FormatInto(format string, args ...any) (out Text) {
	if offsets.FormatInto == nil {
		missing("Format")
	}
	return offsets.FormatInto(format, args)
}

// SetPropertyAt writes one under a key that came from a buffer.
//
//sp:method SetValue
func (p Properties) SetPropertyAt(key Text, value int32) {
	if offsets.SetPropertyAt == nil {
		missing("StringMap.SetValue")
	}
	offsets.SetPropertyAt(p, key, value)
}

// ValueAt reads one under a key that came from a buffer.
//
//sp:method GetValue
func (p Properties) ValueAt(key Text) (found bool, value int32) {
	if offsets.ValueAt == nil {
		missing("StringMap.GetValue")
	}
	return offsets.ValueAt(p, key)
}

// SetOffsetMap writes m_adtOffsets.
//
//sp:globalset m_adtOffsets
func SetOffsetMap(p Properties) {
	if offsets.SetOffsetMap == nil {
		missing("m_adtOffsets")
	}
	offsets.SetOffsetMap(p)
}

// OffsetMap reads it.
//
//sp:global m_adtOffsets
func OffsetMap() Properties {
	if offsets.OffsetMap == nil {
		missing("m_adtOffsets")
	}
	return offsets.OffsetMap()
}

// FindSendPropInfoText is FindSendPropInfo with the prop name in a buffer,
// which is where the gamedata file puts it.
//
//sp:native FindSendPropInfo
func FindSendPropInfoText(class string, prop Text) int32 {
	if offsets.FindSendPropInfoText == nil {
		missing("FindSendPropInfo")
	}
	return offsets.FindSendPropInfoText(class, prop)
}

// StrEqualLiterals compares two literals, which is what a class name test is.
//
//sp:native StrEqual
func StrEqualLiterals(a string, b string) bool {
	if offsets.StrEqualLiterals == nil {
		missing("StrEqual")
	}
	return offsets.StrEqualLiterals(a, b)
}

// ThrowErrorText stops the plugin, saying why, with a buffer among the
// arguments.
//
//sp:native ThrowError
func ThrowErrorText(format string, args ...any) {
	if offsets.ThrowErrorText == nil {
		missing("ThrowError")
	}
	offsets.ThrowErrorText(format, args)
}

// LoadFloatFromAddress reads four bytes as a float, which is what a distance
// stored on a nav area is. The native answers a cell and SourcePawn takes it as
// whatever the caller's type says.
//
//sp:native LoadFromAddress after NumberType_Int32
func LoadFloatFromAddress(address Address) float32 {
	if offsets.LoadFloatFromAddress == nil {
		missing("LoadFromAddress")
	}
	return offsets.LoadFloatFromAddress(address)
}

// AddressOfArea is the nav area read as a memory address, which is what an
// offset is added to.
//
//sp:cast Address
func AddressOfArea(area NavArea) Address {
	return Address(area)
}
