package engine

/*
Reading a map's config file: the KeyValues walk, and the file the bot names
come from.
*/

// MapConfigCalls are the answers.
type MapConfigCalls struct {
	GoBack                  func(kv KeyValues)
	Vector                  func(kv KeyValues, key string) [3]float32
	GotoFirstSubKeyKeysOnly func(kv KeyValues, keysOnly bool) bool
	GotoNextKeyKeysOnly     func(kv KeyValues, keysOnly bool) bool
	StringOr                func(kv KeyValues, key string) Text
	BuildPath               func(format string, args []any) Text
	OpenFile                func(path Text, mode string) File
	ReadFileLine            func(f File) (bool, Text)
	CloseFile               func(f File)
	TrimString              func(text Text)
	PushStringText          func(l List, text Text)
}

var mapConfigs MapConfigCalls

// InstallMapConfigs puts a set of answers behind them.
func InstallMapConfigs(c MapConfigCalls) func() {
	previous := mapConfigs
	mapConfigs = c
	return func() { mapConfigs = previous }
}

// GoBack steps out of the section the last JumpToKey or GotoFirstSubKey went
// into.
//
//sp:method GoBack
func (kv KeyValues) GoBack() {
	if mapConfigs.GoBack == nil {
		missing("KeyValues.GoBack")
	}
	mapConfigs.GoBack(kv)
}

// Vector reads three floats under one key.
//
//sp:method GetVector
func (kv KeyValues) Vector(key string) (out [3]float32) {
	if mapConfigs.Vector == nil {
		missing("KeyValues.GetVector")
	}
	return mapConfigs.Vector(kv, key)
}

// GotoFirstSubKeyKeysOnly is GotoFirstSubKey with the keys-only flag written
// out: false visits values as well as sections, which is how a list of
// spots is walked.
//
//sp:method GotoFirstSubKey
func (kv KeyValues) GotoFirstSubKeyKeysOnly(keysOnly bool) bool {
	if mapConfigs.GotoFirstSubKeyKeysOnly == nil {
		missing("KeyValues.GotoFirstSubKey")
	}
	return mapConfigs.GotoFirstSubKeyKeysOnly(kv, keysOnly)
}

// GotoNextKeyKeysOnly is the same for GotoNextKey.
//
//sp:method GotoNextKey
func (kv KeyValues) GotoNextKeyKeysOnly(keysOnly bool) bool {
	if mapConfigs.GotoNextKeyKeysOnly == nil {
		missing("KeyValues.GotoNextKey")
	}
	return mapConfigs.GotoNextKeyKeysOnly(kv, keysOnly)
}

// StringOr is GetString with the empty default written out, which is what the
// config loaders pass.
//
//sp:method GetString sized after ""
func (kv KeyValues) StringOr(key string) (out Text) {
	if mapConfigs.StringOr == nil {
		missing("KeyValues.GetString")
	}
	return mapConfigs.StringOr(kv, key)
}

// BuildPath is SourceMod's path under addons/sourcemod, formatted.
//
//sp:native BuildPath fills before Path_SM
func BuildPath(format string, args ...any) (out Text) {
	if mapConfigs.BuildPath == nil {
		missing("BuildPath")
	}
	return mapConfigs.BuildPath(format, args)
}

// File is SourceMod's File handle.
//
//sp:tag File
type File int32

// NoFile is null, what OpenFile returns for a path that is not there.
//
//sp:global null
func NoFile() File { return 0 }

// OpenFile opens one for reading or writing. The caller owns it.
//
//sp:native OpenFile
func OpenFile(path Text, mode string) File {
	if mapConfigs.OpenFile == nil {
		missing("OpenFile")
	}
	return mapConfigs.OpenFile(path, mode)
}

// ReadFileLine reads the next line, and says whether there was one. The
// native form, which is what the plugin writes, rather than the methodmap's
// ReadLine.
//
//sp:native ReadFileLine sized
func ReadFileLine(f File) (ok bool, line Text) {
	if mapConfigs.ReadFileLine == nil {
		missing("ReadFileLine")
	}
	return mapConfigs.ReadFileLine(f)
}

// Close releases the handle, which the emitter writes as the plain delete
// statement at every way out.
//
//sp:delete Close
func (f File) Close() {
	if mapConfigs.CloseFile == nil {
		missing("delete File")
	}
	mapConfigs.CloseFile(f)
}

// TrimString strips whitespace from both ends, in place.
//
//sp:native TrimString inplace
func TrimString(text Text) {
	if mapConfigs.TrimString == nil {
		missing("TrimString")
	}
	mapConfigs.TrimString(text)
}

// PushStringText is PushString handed a buffer rather than a literal.
//
//sp:method PushString
func (l List) PushStringText(text Text) {
	if mapConfigs.PushStringText == nil {
		missing("ArrayList.PushString")
	}
	mapConfigs.PushStringText(l, text)
}

// BotNames is m_adtBotNames, the list the random names are drawn from. The
// plugin declares it; this reads it.
//
//sp:global m_adtBotNames
func BotNames() List { return 0 }
