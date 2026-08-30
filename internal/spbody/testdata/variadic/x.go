/*
Package variadic is the printing shape.

A body cannot declare a variadic function and cannot spread into one. It can
call an extern that is variadic, because the arguments are written out at the
call site and go where they were written.
*/
package variadic

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Report is the running commentary a behaviour writes when the debug switch is
// on, which is also the property read.
func Report(actor int32, spot [3]float32) {
	if !engine.DebugActions().Bool() {
		return
	}

	engine.PrintToServer("ConfiguredDispenserSpot: %N takes the named spot %.0f %.0f %.0f",
		actor, spot[0], spot[1], spot[2])
}

// Quiet takes no arguments at all, which is most of them.
func Quiet() {
	engine.PrintToServer("CTFBotMvMEngineerIdle_Update: RELOCATE NEST")
}

// Stack fills a text buffer, which SourceMod declares with its length after it.
func Stack(client int32) {
	stack := engine.ActionStackOf(client)

	engine.PrintToServer("[defenderbots] %N is running %s", client, stack)
}
