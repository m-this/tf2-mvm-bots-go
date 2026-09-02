package engine

/*
Reading a map's config file: the KeyValues walk, and the file the bot names
come from.
*/

// MapConfigCalls are the answers.
type MapConfigCalls struct {
	GoBack                   func(kv KeyValues)
	Vector                   func(kv KeyValues, key string) [3]float32
	GotoFirstSubKeyKeysOnly  func(kv KeyValues, keysOnly bool) bool
	GotoNextKeyKeysOnly      func(kv KeyValues, keysOnly bool) bool
	StringOr                 func(kv KeyValues, key string) Text
	BuildPath                func(format string, args []any) Text
	OpenFile                 func(path Text, mode string) File
	ReadFileLine             func(f File) (bool, Text)
	CloseFile                func(f File)
	TrimString               func(text Text)
	PushStringText           func(l List, text Text)
	SetClientName            func(client int32, name Text)
	DoesAnyPlayerUseThisName func(name Text) bool
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

// SetClientName renames a player, which the server tells everybody about.
//
//sp:native SetClientName
func SetClientName(client int32, name Text) {
	if mapConfigs.SetClientName == nil {
		missing("SetClientName")
	}
	mapConfigs.SetClientName(client, name)
}

// DoesAnyPlayerUseThisName walks the players for one already called that.
// Ported, stocks.
//
//sp:body DoesAnyPlayerUseThisName
func DoesAnyPlayerUseThisName(name Text) bool {
	if mapConfigs.DoesAnyPlayerUseThisName == nil {
		missing("DoesAnyPlayerUseThisName")
	}
	return mapConfigs.DoesAnyPlayerUseThisName(name)
}

/*
The map file and the mission's difficulty.

Both are read once a map has loaded and neither is a decision: the file says
what it says, and the difficulty comes off the popfile's name.
*/

// MapConfigMapCalls are the answers for those.
type MapConfigMapCalls struct {
	ResetMapConfig            func()
	StringInto                func(kv KeyValues, key string, out Text, maxlen int32, def string)
	SetMovingNests            func(on bool)
	MvMPopfileName            func(resource int32) Text
	ReplaceString             func(text Text, maxlen int32, search string, replace string)
	MapConfigCounts           func() [7]int32
	MovingNestsIsSet          func() bool
	BuildPathText             func(format Text) Text
	CloseHandle               func(kv KeyValues)
	MissionDifficultyFilePath func(difficulty MissionDifficulty) Text
}

var mapConfigMaps MapConfigMapCalls

// InstallMapConfigMaps puts a set of answers behind them.
func InstallMapConfigMaps(c MapConfigMapCalls) func() {
	previous := mapConfigMaps
	mapConfigMaps = c
	return func() { mapConfigMaps = previous }
}

// ResetMapConfig empties every list on the record and clears the two plain
// fields, which is the enum struct's own method.
//
//sp:plugin g_arrMapConfig.Reset
func ResetMapConfig() {
	if mapConfigMaps.ResetMapConfig == nil {
		missing("g_arrMapConfig.Reset")
	}
	mapConfigMaps.ResetMapConfig()
}

// StringInto is GetString written out: the destination is a buffer that
// already exists, which the sized form cannot express because it makes one.
//
//sp:method GetString
func (kv KeyValues) StringInto(key string, out Text, maxlen int32, def string) {
	if mapConfigMaps.StringInto == nil {
		missing("KeyValues.GetString")
	}
	mapConfigMaps.StringInto(kv, key, out, maxlen, def)
}

// MapCompositionSize is sizeof(g_arrMapConfig.strComposition).
//
//sp:global sizeof(g_arrMapConfig.strComposition)
func MapCompositionSize() int32 { return 128 }

// SetMovingNests writes whether this map's nests move.
//
//sp:globalset g_arrMapConfig.bMovingNests
func SetMovingNests(on bool) {
	if mapConfigMaps.SetMovingNests == nil {
		missing("g_arrMapConfig.bMovingNests")
	}
	mapConfigMaps.SetMovingNests(on)
}

// MovingNests says whether they do.
//
//sp:global g_arrMapConfig.bMovingNests
func MovingNests() bool {
	if mapConfigMaps.MovingNestsIsSet == nil {
		missing("g_arrMapConfig.bMovingNests")
	}
	return mapConfigMaps.MovingNestsIsSet()
}

// MvMPopfileName is the mission being played, path and extension included.
//
//sp:native TF2_GetMvMPopfileName sized
func MvMPopfileName(resource int32) (name Text) {
	if mapConfigMaps.MvMPopfileName == nil {
		missing("TF2_GetMvMPopfileName")
	}
	return mapConfigMaps.MvMPopfileName(resource)
}

// ReplaceString rewrites every occurrence in place.
//
//sp:native ReplaceString
func ReplaceString(text Text, maxlen int32, search string, replace string) {
	if mapConfigMaps.ReplaceString == nil {
		missing("ReplaceString")
	}
	mapConfigMaps.ReplaceString(text, maxlen, search, replace)
}

/*
The difficulty a mission is, and the files that name them.

internal/body/shared declares the enum and the table on the SourcePawn side. A
body may only import this package, so both are reached through it.
*/

// MissionDifficulty is eMissionDifficulty.
//
//sp:tag eMissionDifficulty
type MissionDifficulty int32

// MissionUnknown is the answer for a popfile nothing recognises.
//
//sp:global MISSION_UNKNOWN
func MissionUnknown() MissionDifficulty { return 0 }

// MissionNormal is the first real one, which is where the search starts.
//
//sp:global MISSION_NORMAL
func MissionNormal() MissionDifficulty { return 1 }

// MissionIntermediate is the second.
//
//sp:global MISSION_INTERMEDIATE
func MissionIntermediate() MissionDifficulty { return 2 }

// MissionAdvanced is the third.
//
//sp:global MISSION_ADVANCED
func MissionAdvanced() MissionDifficulty { return 3 }

// MissionExpert is the fourth.
//
//sp:global MISSION_EXPERT
func MissionExpert() MissionDifficulty { return 4 }

// MissionNightmare is the fifth, which no official mission uses.
//
//sp:global MISSION_NIGHTMARE
func MissionNightmare() MissionDifficulty { return 5 }

// MissionMaxCount is one past the last, which is where the search stops.
//
//sp:global MISSION_MAX_COUNT
func MissionMaxCount() MissionDifficulty { return 6 }

// MissionDifficultyFilePath is the file listing the missions of that
// difficulty, relative to SourceMod's own directory.
//
//sp:slot g_sMissionDifficultyFilePaths
func MissionDifficultyFilePath(difficulty MissionDifficulty) Text {
	if mapConfigMaps.MissionDifficultyFilePath == nil {
		missing("g_sMissionDifficultyFilePaths")
	}
	return mapConfigMaps.MissionDifficultyFilePath(difficulty)
}

// CloseHandle releases a handle by the older spelling. The map config writes
// it rather than delete, and a port does not change what runs.
//
//sp:native CloseHandle frees
func CloseHandle(kv KeyValues) {
	if mapConfigMaps.CloseHandle == nil {
		missing("CloseHandle")
	}
	mapConfigMaps.CloseHandle(kv)
}

// PathMax is PLATFORM_MAX_PATH, the length of every path buffer the plugin
// declares.
//
//sp:global PLATFORM_MAX_PATH
func PathMax() int32 { return 256 }

// BuildPathText is BuildPath handed a buffer rather than a literal, which is
// how the difficulty files are named: the path comes off a table.
//
//sp:native BuildPath fills before Path_SM
func BuildPathText(format Text) (out Text) {
	if mapConfigMaps.BuildPathText == nil {
		missing("BuildPath")
	}
	return mapConfigMaps.BuildPathText(format)
}
