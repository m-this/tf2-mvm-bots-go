package engine

/*
The bot state every behaviour shares.

nextbot_behavior.sp keeps it as arrays over client slots, and a behaviour both
reads and writes them. Reached here as a read and a write rather than owned,
because the file that declares them is not ported yet and one array cannot live
in two places.

Every one of these goes when nextbot_behavior.sp moves: they become ordinary
package state in the Go that owns them, and the two declarations here collapse
into a subscript the generator writes itself.
*/

// StateCalls are the shared arrays, as answers a test installs.
type StateCalls struct {
	Path          func(actor int32) Path
	RepathTime    func(actor int32) float32
	SetRepathTime func(actor int32, when float32)
}

var state StateCalls

// InstallState puts a set of answers behind the shared arrays.
func InstallState(c StateCalls) func() {
	previous := state
	state = c
	return func() { state = previous }
}

// Path is CBaseNPC's PathFollower, the route a bot is walking.
//
//sp:tag PathFollower
type Path int32

// PathOf is the route the bot is walking, which every behaviour updates.
//
//sp:slot m_pPath
func PathOf(actor int32) Path {
	if state.Path == nil {
		missing("m_pPath")
	}
	return state.Path(actor)
}

// RepathTime is when the bot may next ask the game for a route. Asking every
// frame is what killed a server in mvm-bj8.
//
//sp:slot m_flRepathTime
func RepathTime(actor int32) float32 {
	if state.RepathTime == nil {
		missing("m_flRepathTime")
	}
	return state.RepathTime(actor)
}

// SetRepathTime puts the next repath off until then.
//
//sp:slotset m_flRepathTime
func SetRepathTime(actor int32, when float32) {
	if state.SetRepathTime == nil {
		missing("m_flRepathTime")
	}
	state.SetRepathTime(actor, when)
}

/*
	SetMinLookAheadDistance is how far along the path the bot aims

Nothing happens here, and nothing can: a path is a CBaseNPC object living in the
game's memory and there is no answer to install that would mean anything. The
declaration exists so a body can call it and the generator can emit it, and the
differential test never walks one, because a behaviour cannot be run under
spshell at all.

//sp:method SetMinLookAheadDistance
*/
//nolint:revive // unused-parameter: the signature is SourceMod's, not ours
func (p Path) SetMinLookAheadDistance(distance float32) {
	if state.Path == nil {
		missing("PathFollower.SetMinLookAheadDistance")
	}
}

// Update walks the bot one step along the path. Nothing here either, for the
// reason above.
//
//sp:method Update
//nolint:revive // unused-parameter: the signature is SourceMod's, not ours
func (p Path) Update(bot Bot) {
	if state.Path == nil {
		missing("PathFollower.Update")
	}
}
