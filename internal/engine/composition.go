package engine

/*
The named team: the lineup a server owner typed, and the classes they banned.

The convar wins over the map. Somebody who typed a team into the console is
answering a question the map file guessed at, and the map is a default rather
than an instruction.
*/

// CompositionCalls are the answers.
type CompositionCalls struct {
	ConVarStringInto         func(c ConVar, out Text, maxlen int32)
	ConVarString             func(c ConVar) Text
	ClassBlacklist           func() ConVar
	StrcopyFromText          func(out Text, maxlen int32, from Text)
	ClearBuildingsBeforeKick func(client int32)
	GetWantedTeamComposition func(out Text, maxlen int32)
	IsBotClassBlacklisted    func(class Text) bool
	IsClassInTeamComposition func(class Text, typedTeamOnly bool) bool
}

var compositions CompositionCalls

// InstallCompositions puts a set of answers behind them.
func InstallCompositions(c CompositionCalls) func() {
	previous := compositions
	compositions = c
	return func() { compositions = previous }
}

// StringInto reads a convar into a buffer the caller already has.
//
//sp:method GetString
func (c ConVar) StringInto(out Text, maxlen int32) {
	if compositions.ConVarStringInto == nil {
		missing("ConVar.GetString")
	}
	compositions.ConVarStringInto(c, out, maxlen)
}

// String reads a convar into a fresh buffer.
//
//sp:method GetString fills
func (c ConVar) String() (out Text) {
	if compositions.ConVarString == nil {
		missing("ConVar.GetString")
	}
	return compositions.ConVarString(c)
}

// ClassBlacklist is redbots_manager_class_blacklist, the classes a server owner
// told the mod never to play.
//
//sp:global redbots_manager_class_blacklist
func ClassBlacklist() ConVar { return 0 }

// StrcopyFromText overwrites a buffer with another buffer.
//
//sp:native strcopy
func StrcopyFromText(out Text, maxlen int32, from Text) {
	if compositions.StrcopyFromText == nil {
		missing("strcopy")
	}
	compositions.StrcopyFromText(out, maxlen, from)
}

// ClearBuildingsBeforeKick takes an engineer's buildings down so they do not
// outlive the bot that owns them. Ported, roster_counts.
//
//sp:body ClearBuildingsBeforeKick
func ClearBuildingsBeforeKick(client int32) {
	if compositions.ClearBuildingsBeforeKick == nil {
		missing("ClearBuildingsBeforeKick")
	}
	compositions.ClearBuildingsBeforeKick(client)
}

// GetWantedTeamComposition is the lineup to fill RED with, or an empty string
// to leave it to the lineup mode. Ported, composition.
//
//sp:body GetWantedTeamComposition
func GetWantedTeamComposition(out Text, maxlen int32) {
	if compositions.GetWantedTeamComposition == nil {
		missing("GetWantedTeamComposition")
	}
	compositions.GetWantedTeamComposition(out, maxlen)
}

// IsBotClassBlacklisted says the server was told never to play that class.
// Ported, composition.
//
//sp:body IsBotClassBlacklisted
func IsBotClassBlacklisted(class Text) bool {
	if compositions.IsBotClassBlacklisted == nil {
		missing("IsBotClassBlacklisted")
	}
	return compositions.IsBotClassBlacklisted(class)
}

// IsClassInTeamComposition asks whether the named team wants that class
// anywhere. Ported, composition.
//
//sp:body IsClassInTeamComposition
func IsClassInTeamComposition(class Text, typedTeamOnly bool) bool {
	if compositions.IsClassInTeamComposition == nil {
		missing("IsClassInTeamComposition")
	}
	return compositions.IsClassInTeamComposition(class, typedTeamOnly)
}

// TextSize is the length of a Text buffer, which is what a call taking an
// explicit maximum is handed for one.
//
//sp:global 512
func TextSize() int32 { return 512 }
