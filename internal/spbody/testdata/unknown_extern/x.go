package x

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Install is a real function in the extern package and carries no //sp:
// directive, so it names no SourcePawn and cannot be called from a body.
func Ask() bool {
	engine.Install(engine.Calls{})

	return true
}
