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

func InstallDirectives(c DirectiveCalls) func() {
	previous := directives
	Fill(&c)
	directives = c
	return func() { directives = previous }
}

//sp:library RegisterDirectiveNatives
func RegisterDirectiveNatives() {
	if directives.Register != nil {
		directives.Register()
	}
}

//sp:library DefenderRallyActive
func DefenderRallyActive(client int32) bool {
	return directives.RallyActive != nil && directives.RallyActive(client)
}

//sp:library DefenderRallyX
func DefenderRallyX(client int32) float32 {
	if directives.RallyX == nil {
		return 0
	}
	return directives.RallyX(client)
}

//sp:library DefenderRallyY
func DefenderRallyY(client int32) float32 {
	if directives.RallyY == nil {
		return 0
	}
	return directives.RallyY(client)
}

//sp:library DefenderRallyZ
func DefenderRallyZ(client int32) float32 {
	if directives.RallyZ == nil {
		return 0
	}
	return directives.RallyZ(client)
}

//sp:library DefenderSeekEnemies
func DefenderSeekEnemies(client int32) bool {
	return directives.SeekEnemies != nil && directives.SeekEnemies(client)
}

//sp:library DefenderBuyAnywhere
func DefenderBuyAnywhere(client int32) bool {
	return directives.BuyAnywhere != nil && directives.BuyAnywhere(client)
}
