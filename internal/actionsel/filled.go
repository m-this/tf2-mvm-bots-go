package actionsel

// SelectFilled is Select with mvm-vnn filled: the scout who has no money, no
// giant, no tank and no target holds the hatch instead of standing still.
//
// It is a candidate fix and it is deliberately not the function the plugin's
// table is generated from, so that turning it on is one switch with one
// measurement behind it rather than something that arrived with the port.
func SelectFilled(state RoundState, class Class, f Flags) Action {
	a := Select(state, class, f)
	if a == ActionStrandedAsShipped {
		return ActionGuardPoint
	}
	return a
}
