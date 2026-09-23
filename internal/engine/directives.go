package engine

// DirectiveCalls are short lived per-defender orders supplied by another plugin.
type DirectiveCalls struct {
	Register    func()
	RallyActive func(int32) bool
	RallyX      func(int32) float32
	RallyY      func(int32) float32
	RallyZ      func(int32) float32
	SeekEnemies func(int32) bool
	BuyAnywhere func(int32) bool
}

var directives DirectiveCalls

// InstallDirectives puts a set of answers behind the directive calls and
// returns the restore.
func InstallDirectives(c DirectiveCalls) func() {
	previous := directives
	Fill(&c)
	directives = c
	return func() { directives = previous }
}

// RegisterDirectiveNatives asks the plugin to expose the directive native.
//
//sp:library RegisterDirectiveNatives
func RegisterDirectiveNatives() {
	if directives.Register != nil {
		directives.Register()
	}
}

// DefenderRallyActive says whether a caller currently wants this defender at
// a rally point. False whenever no order is live, including with no caller.
//
//sp:library DefenderRallyActive
func DefenderRallyActive(client int32) bool {
	return directives.RallyActive != nil && directives.RallyActive(client)
}

// DefenderRallyX is the rally point's x, and zero when no order is live.
//
//sp:library DefenderRallyX
func DefenderRallyX(client int32) float32 {
	if directives.RallyX == nil {
		return 0
	}
	return directives.RallyX(client)
}

// DefenderRallyY is the rally point's y, and zero when no order is live.
//
//sp:library DefenderRallyY
func DefenderRallyY(client int32) float32 {
	if directives.RallyY == nil {
		return 0
	}
	return directives.RallyY(client)
}

// DefenderRallyZ is the rally point's z, and zero when no order is live.
//
//sp:library DefenderRallyZ
func DefenderRallyZ(client int32) float32 {
	if directives.RallyZ == nil {
		return 0
	}
	return directives.RallyZ(client)
}

// DefenderSeekEnemies says whether a caller wants this defender hunting rather
// than holding its usual post.
//
//sp:library DefenderSeekEnemies
func DefenderSeekEnemies(client int32) bool {
	return directives.SeekEnemies != nil && directives.SeekEnemies(client)
}

// DefenderBuyAnywhere says whether this defender may shop without walking to
// an upgrade station first.
//
//sp:library DefenderBuyAnywhere
func DefenderBuyAnywhere(client int32) bool {
	return directives.BuyAnywhere != nil && directives.BuyAnywhere(client)
}
