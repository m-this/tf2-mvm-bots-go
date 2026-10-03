package hooks

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
A MainAction is not started when it is built, and the plugin must not ask the bot
anything until it is (mvm-9q4). The game builds one at every spawn and every
intention reset, so a bot that was started before is not started now.
*/
func TestAMainActionIsStartedOnlyOnceItHasRun(t *testing.T) {
	const actor = 9

	var hooked []string
	restore := engine.Install(engine.Calls{MaxClients: func() int32 { return 64 }})
	defer restore()
	restoreLogs := engine.InstallTextOps(engine.TextOps{
		StrEqualLiteral: func(a, b string, _ bool) bool { return a == b },
	})
	defer restoreLogs()
	restoreActions := engine.InstallActions(engine.ActionCalls{
		SetCallback:      func(_ engine.Behaviour, slot string) { hooked = append(hooked, slot) },
		ShouldEmptyStack: func(int32) bool { return false },
	})
	defer restoreActions()
	restoreFronts := engine.InstallFronts(engine.FrontCalls{
		DefenderBotFlag: func(int32) bool { return true },
	})
	defer restoreFronts()
	defer ForgetMainActionStart(actor)

	mainActionStarted[actor] = true // the last behaviour this bot had

	OnActionCreated(1, actor, "MainAction")
	if MainActionStarted(actor) {
		t.Fatal("a MainAction just built reads as started")
	}
	onStart := false
	for _, slot := range hooked {
		onStart = onStart || slot == "OnStart"
	}
	if !onStart {
		t.Fatalf("MainAction was given %v and no OnStart, so nothing would ever mark it started", hooked)
	}

	MainActionOnStart(1, actor, 0, 0)
	if !MainActionStarted(actor) {
		t.Fatal("a MainAction that has run its OnStart does not read as started")
	}

	ForgetMainActionStart(actor)
	if MainActionStarted(actor) {
		t.Fatal("a reset behaviour still reads as started")
	}

	// A bot already running when the plugin loaded is never shown an OnStart.
	MainActionUpdate(1, actor, 0.1, 0)
	if !MainActionStarted(actor) {
		t.Fatal("a MainAction that is updating does not read as started")
	}
}
