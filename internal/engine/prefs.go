package engine

/*
The player preference file and the server loadout, both of which are KeyValues,
and the Steam ID a preference is filed under.
*/

// PrefCalls are the answers.
type PrefCalls struct {
	JumpToKey             func(kv KeyValues, key string, create bool) bool
	JumpToKeyText         func(kv KeyValues, key Text, create bool) bool
	Rewind                func(kv KeyValues)
	KVNum                 func(kv KeyValues, key string, fallback int32) int32
	KVNumText             func(kv KeyValues, key Text, fallback int32) int32
	SetNum                func(kv KeyValues, key string, value int32)
	SetNumText            func(kv KeyValues, key Text, value int32)
	KVString              func(kv KeyValues, key string) Text
	SectionName           func(kv KeyValues) Text
	GotoFirstSubKey       func(kv KeyValues) bool
	GotoNextKey           func(kv KeyValues) bool
	ClientAuthID          func(client int32, kind AuthIDKind) (bool, Text)
	StringToInt           func(text Text) int32
	IntToString           func(value int32) (int32, Text)
	PushString            func(l List, text string)
	NewStringList         func(blockSize int32) List
	ClientTeam            func(client int32) Team
	AddDefenderTFBot      func(count int32, class Text, team string, difficulty string)
	AddRandomDefenderBots func(amount int32)
	RandomWeaponForClass  func(class string, slot string) int32
	IsServerFull          func() bool
}

var prefs PrefCalls

// InstallPrefs puts a set of answers behind them.
func InstallPrefs(c PrefCalls) func() {
	previous := prefs
	prefs = c
	return func() { prefs = previous }
}

// NoKeyValues is null, which is what a config the server did not write reads as.
//
//sp:global null
func NoKeyValues() KeyValues { return 0 }

// JumpToKey stands on a key, making it when asked to.
//
//sp:method JumpToKey
func (kv KeyValues) JumpToKey(key string, create bool) bool {
	if prefs.JumpToKey == nil {
		missing("KeyValues.JumpToKey")
	}
	return prefs.JumpToKey(kv, key, create)
}

// JumpToKeyText is the same for a key that was built rather than written out.
//
//sp:method JumpToKey
func (kv KeyValues) JumpToKeyText(key Text, create bool) bool {
	if prefs.JumpToKeyText == nil {
		missing("KeyValues.JumpToKey")
	}
	return prefs.JumpToKeyText(kv, key, create)
}

// Rewind goes back to the root.
//
//sp:method Rewind
func (kv KeyValues) Rewind() {
	if prefs.Rewind == nil {
		missing("KeyValues.Rewind")
	}
	prefs.Rewind(kv)
}

// Num is a number under the current key, or the fallback.
//
//sp:method GetNum
func (kv KeyValues) Num(key string, fallback int32) int32 {
	if prefs.KVNum == nil {
		missing("KeyValues.GetNum")
	}
	return prefs.KVNum(kv, key, fallback)
}

// NumText is the same for a key that was handed in.
//
//sp:method GetNum
func (kv KeyValues) NumText(key Text, fallback int32) int32 {
	if prefs.KVNumText == nil {
		missing("KeyValues.GetNum")
	}
	return prefs.KVNumText(kv, key, fallback)
}

// SetNum writes one.
//
//sp:method SetNum
func (kv KeyValues) SetNum(key string, value int32) {
	if prefs.SetNum == nil {
		missing("KeyValues.SetNum")
	}
	prefs.SetNum(kv, key, value)
}

// SetNumText writes one under a key that was handed in.
//
//sp:method SetNum
func (kv KeyValues) SetNumText(key Text, value int32) {
	if prefs.SetNumText == nil {
		missing("KeyValues.SetNum")
	}
	prefs.SetNumText(kv, key, value)
}

// String is text under the current key.
//
//sp:method GetString sized
func (kv KeyValues) String(key string) (out Text) {
	if prefs.KVString == nil {
		missing("KeyValues.GetString")
	}
	return prefs.KVString(kv, key)
}

// SectionName is the name of the key it is standing on.
//
//sp:method GetSectionName sized
func (kv KeyValues) SectionName() (out Text) {
	if prefs.SectionName == nil {
		missing("KeyValues.GetSectionName")
	}
	return prefs.SectionName(kv)
}

// GotoFirstSubKey steps into the first child.
//
//sp:method GotoFirstSubKey
func (kv KeyValues) GotoFirstSubKey() bool {
	if prefs.GotoFirstSubKey == nil {
		missing("KeyValues.GotoFirstSubKey")
	}
	return prefs.GotoFirstSubKey(kv)
}

// GotoNextKey steps to the next sibling.
//
//sp:method GotoNextKey
func (kv KeyValues) GotoNextKey() bool {
	if prefs.GotoNextKey == nil {
		missing("KeyValues.GotoNextKey")
	}
	return prefs.GotoNextKey(kv)
}

// ClientAuthID is the Steam ID a preference is filed under.
//
//sp:native GetClientAuthId sized
func ClientAuthID(client int32, kind AuthIDKind) (ok bool, id Text) {
	if prefs.ClientAuthID == nil {
		missing("GetClientAuthId")
	}
	return prefs.ClientAuthID(client, kind)
}

// StringToInt reads a number out of text.
//
//sp:native StringToInt
func StringToInt(text Text) int32 {
	if prefs.StringToInt == nil {
		missing("StringToInt")
	}
	return prefs.StringToInt(text)
}

// IntToString writes one into a buffer.
//
//sp:native IntToString sized
func IntToString(value int32) (written int32, out Text) {
	if prefs.IntToString == nil {
		missing("IntToString")
	}
	return prefs.IntToString(value)
}

// PushString adds text to a list of text.
//
//sp:method PushString
func (l List) PushString(text string) {
	if prefs.PushString == nil {
		missing("ArrayList.PushString")
	}
	prefs.PushString(l, text)
}

// NewStringList makes a list whose cells hold text rather than a number.
//
//sp:new ArrayList
func NewStringList(blockSize int32) List {
	if prefs.NewStringList == nil {
		missing("new ArrayList")
	}
	return prefs.NewStringList(blockSize)
}

// ClientTeam is the team as TFTeam rather than as a number.
//
//sp:native TF2_GetClientTeam
func ClientTeam(client int32) Team {
	if prefs.ClientTeam == nil {
		missing("TF2_GetClientTeam")
	}
	return prefs.ClientTeam(client)
}

/*
The two KeyValues the plugin loads and this reads.

Both are globals rather than handles this code opens: player_pref.sp still owns
the loading, and a config the server did not write reads as null.
*/

// PluginPrefix is PLUGIN_PREFIX, what the mod says before it says anything.
//
//sp:global PLUGIN_PREFIX
func PluginPrefix() string { return "" }

// AddDefenderTFBotOf adds bots of one class to a team at one difficulty.
//
//sp:plugin AddDefenderTFBot
func AddDefenderTFBotOf(count int32, class Text, team string, difficulty string) {
	if prefs.AddDefenderTFBot == nil {
		missing("AddDefenderTFBot")
	}
	prefs.AddDefenderTFBot(count, class, team, difficulty)
}

// AddRandomDefenderBots adds that many, class unchosen.
//
//sp:plugin AddRandomDefenderBots
func AddRandomDefenderBots(amount int32) {
	if prefs.AddRandomDefenderBots == nil {
		missing("AddRandomDefenderBots")
	}
	prefs.AddRandomDefenderBots(amount)
}

// RandomWeaponForClass draws one out of the pool. Ported, loadouts.sp.
//
//sp:body GetRandomWeaponForClass
func RandomWeaponForClass(class string, slot string) int32 {
	if prefs.RandomWeaponForClass == nil {
		missing("GetRandomWeaponForClass")
	}
	return prefs.RandomWeaponForClass(class, slot)
}

// IsServerFull says there is no room for another bot. Ported, util.sp.
//
//sp:body IsServerFull
func IsServerFull() bool {
	if prefs.IsServerFull == nil {
		missing("IsServerFull")
	}
	return prefs.IsServerFull()
}

// AuthIDKind is SourceMod's AuthIdType, which spelling of a Steam id is wanted.
//
//sp:tag AuthIdType
type AuthIDKind int32

// AuthIDSteam3 is AuthId_Steam3, the [U:1:...] form the preference file is
// keyed by.
//
//sp:global AuthId_Steam3
func AuthIDSteam3() AuthIDKind { return 2 }
