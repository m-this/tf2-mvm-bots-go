package engine

/*
Turning what somebody typed into the players they meant.

SourceMod owns the pattern language, so this is one native and the error
message that goes with it.
*/

// TargetingCalls are the answers.
type TargetingCalls struct {
	ProcessTargetString func(pattern Text, admin int32, targets [101]int32, maxTargets int32, filterFlags int32, targetName Text, nameMax int32, isML bool) int32
	ReplyToTargetError  func(client int32, reason int32)
	ShowPlayerUpgrades  func(client int32, target int32, slot int32)
}

var targetings TargetingCalls

// InstallTargetings puts a set of answers behind them.
func InstallTargetings(c TargetingCalls) func() {
	previous := targetings
	Fill(&c)
	targetings = c
	return func() { targetings = previous }
}

// MaxTargets is MAXPLAYERS, the most a pattern can name.
//
//sp:global MAXPLAYERS
func MaxTargets() int32 { return 101 }

// CommandFilterAlive is COMMAND_FILTER_ALIVE, which drops the dead from the
// answer.
//
//sp:global COMMAND_FILTER_ALIVE
func CommandFilterAlive() int32 { return 1 }

/*
ProcessTargetString is who the pattern means, and how many.

Three of its arguments are written through rather than read: the targets, the
name the pattern resolved to and whether that name is a translation phrase.
SourcePawn takes them by reference, so passing the local is what fills it, and
they sit between ordinary arguments rather than at either end.

//sp:native ProcessTargetString
*/
func ProcessTargetString(pattern Text, admin int32, targets [101]int32, maxTargets int32, filterFlags int32, targetName Text, nameMax int32, isML bool) int32 {
	return targetings.ProcessTargetString(pattern, admin, targets, maxTargets, filterFlags, targetName, nameMax, isML)
}

// ReplyToTargetError says why a pattern named nobody, in the caller's own
// language.
//
//sp:native ReplyToTargetError
func ReplyToTargetError(client int32, reason int32) { targetings.ReplyToTargetError(client, reason) }

// ShowPlayerUpgrades prints one player's, either the player itself or one
// slot. Ported, upgradereport.
//
//sp:body ShowPlayerUpgrades
func ShowPlayerUpgrades(client int32, target int32, slot int32) {
	targetings.ShowPlayerUpgrades(client, target, slot)
}
