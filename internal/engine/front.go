package engine

/*
Taking up a position before the wave, and the text the debug command prints.

Format and strcopy write into a buffer whose length comes first, which is the
other of the two text shapes: GetEntityClassname reads into one whose length
comes last.
*/

// FrontCalls are the answers.
type FrontCalls struct {
	Format                               func(format string, args ...any) Text
	TextFrom                             func(source string) Text
	EntPropString                        func(entity int32, propType PropType, prop string) Text
	RawPlayerClassName                   func(class Class) Text
	IsStuck                              func(l Locomotion) bool
	ClearStuckStatus                     func(l Locomotion, reason string)
	SetPlayerReady                       func(client int32, ready bool)
	IsPlayerReady                        func(client int32) bool
	StuckCountOf                         func(client int32) int32
	PathFailuresOf                       func(client int32) int32
	RecoverDefenderFromDisconnectedSpawn func(actor int32)
	LookupEntityActionByName             func(client int32, name string) int32
	IsDefenderBot                        func(client int32) bool
	DefenderBotFlag                      func(client int32) bool
	ShoppedThisBreak                     func(client int32) bool
}

var fronts FrontCalls

// InstallFronts puts a set of answers behind them.
func InstallFronts(c FrontCalls) func() {
	previous := fronts
	fronts = c
	return func() { fronts = previous }
}

// FeatureHoldTheNest is the switch for the ranged classes waiting at the nest
// rather than at the gate.
//
//sp:global FEATURE_HOLD_THE_NEST
func FeatureHoldTheNest() int32 { return 4 }

// InvalidAction is INVALID_ACTION.
//
//sp:global INVALID_ACTION
func InvalidAction() int32 { return 0 }

// Format writes text into a buffer whose length comes first.
//
//sp:native Format fills
func Format(format string, args ...any) (out Text) {
	if fronts.Format == nil {
		missing("Format")
	}
	return fronts.Format(format, args...)
}

// TextFrom is strcopy, which is Format with nothing to substitute.
//
//sp:native strcopy fills
func TextFrom(source string) (out Text) {
	if fronts.TextFrom == nil {
		missing("strcopy")
	}
	return fronts.TextFrom(source)
}

// EntPropString reads a string property into a buffer whose length comes last.
//
//sp:native GetEntPropString sized
func EntPropString(entity int32, propType PropType, prop string) (out Text) {
	if fronts.EntPropString == nil {
		missing("GetEntPropString")
	}
	return fronts.EntPropString(entity, propType, prop)
}

// RawPlayerClassName is the class name the debug output prints.
//
//sp:slot g_sRawPlayerClassNames
func RawPlayerClassName(class Class) Text {
	if fronts.RawPlayerClassName == nil {
		missing("g_sRawPlayerClassNames")
	}
	return fronts.RawPlayerClassName(class)
}

// IsStuck says the bot is walking on the spot, which locomotion already knows
// and nothing outside the engineer has ever asked it.
//
//sp:method IsStuck
func (l Locomotion) IsStuck() bool {
	if fronts.IsStuck == nil {
		missing("ILocomotion.IsStuck")
	}
	return fronts.IsStuck(l)
}

// ClearStuckStatus tells it the bot has been dealt with.
//
//sp:method ClearStuckStatus
func (l Locomotion) ClearStuckStatus(reason string) {
	if fronts.ClearStuckStatus == nil {
		missing("ILocomotion.ClearStuckStatus")
	}
	fronts.ClearStuckStatus(l, reason)
}

// SetPlayerReady presses the ready button for the bot.
//
//sp:body SetPlayerReady
func SetPlayerReady(client int32, ready bool) {
	if fronts.SetPlayerReady == nil {
		missing("SetPlayerReady")
	}
	fronts.SetPlayerReady(client, ready)
}

// IsPlayerReady says whether the ready registered.
//
//sp:body IsPlayerReady
func IsPlayerReady(client int32) bool {
	if fronts.IsPlayerReady == nil {
		missing("IsPlayerReady")
	}
	return fronts.IsPlayerReady(client)
}

// StuckCountOf is how many times the bot has been wedged this break.
//
//sp:plugin StuckCountOf
func StuckCountOf(client int32) int32 {
	if fronts.StuckCountOf == nil {
		missing("StuckCountOf")
	}
	return fronts.StuckCountOf(client)
}

// PathFailuresOf is how many routes it has been refused.
//
//sp:plugin PathFailuresOf
func PathFailuresOf(client int32) int32 {
	if fronts.PathFailuresOf == nil {
		missing("PathFailuresOf")
	}
	return fronts.PathFailuresOf(client)
}

// RecoverDefenderFromDisconnectedSpawn puts a bot back on the mesh when the
// spawn it woke in has no way out.
//
//sp:plugin RecoverDefenderFromDisconnectedSpawn
func RecoverDefenderFromDisconnectedSpawn(actor int32) {
	if fronts.RecoverDefenderFromDisconnectedSpawn == nil {
		missing("RecoverDefenderFromDisconnectedSpawn")
	}
	fronts.RecoverDefenderFromDisconnectedSpawn(actor)
}

// LookupEntityActionByName is the action of that name the bot is running, and
// INVALID_ACTION when it is not.
//
//sp:native ActionsManager.LookupEntityActionByName
func LookupEntityActionByName(client int32, name string) int32 {
	if fronts.LookupEntityActionByName == nil {
		missing("ActionsManager.LookupEntityActionByName")
	}
	return fronts.LookupEntityActionByName(client, name)
}

// DefenderBotFlag is the flag itself, which is what a caller reading
// g_bIsDefenderBot[client] asks.
//
//sp:slot g_bIsDefenderBot
func DefenderBotFlag(client int32) bool {
	if fronts.DefenderBotFlag == nil {
		missing("g_bIsDefenderBot")
	}
	return fronts.DefenderBotFlag(client)
}

/*
IsDefenderBot is the function, which is not the same question as the flag.

It answers yes for a fake client whose name carries the mod's identity even
though nothing has flagged him yet, which is how a bot the manager has not
adopted is still recognised. A port that reads the flag where the plugin called
the function quietly narrows it.

//sp:plugin IsDefenderBot
*/
func IsDefenderBot(client int32) bool {
	if fronts.IsDefenderBot == nil {
		missing("IsDefenderBot")
	}
	return fronts.IsDefenderBot(client)
}

// ShoppedThisBreak says the bot has been to the upgrade station.
//
//sp:slot g_bShoppedThisBreak
func ShoppedThisBreak(client int32) bool {
	if fronts.ShoppedThisBreak == nil {
		missing("g_bShoppedThisBreak")
	}
	return fronts.ShoppedThisBreak(client)
}
