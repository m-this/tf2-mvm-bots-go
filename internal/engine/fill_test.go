package engine_test

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// TestAnUninstalledAnswerStillPanicsByName is the property the nil check used
// to give at every call site, kept while the call sites lost it.
func TestAnUninstalledAnswerStillPanicsByName(t *testing.T) {
	defer engine.InstallAmmo(engine.AmmoCalls{})()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a call with no answer behind it did not panic")
		}
		if !strings.Contains(r.(string), "AmmoCalls.IsAmmoFull") {
			t.Errorf("the panic does not name the call: %v", r)
		}
		if !strings.Contains(r.(string), "no answer is installed") {
			t.Errorf("the panic does not say why: %v", r)
		}
	}()
	engine.IsAmmoFull(1)
}

// TestAnInstalledAnswerIsTheOneCalled is the other half: filling the nils must
// not overwrite what the caller did install.
func TestAnInstalledAnswerIsTheOneCalled(t *testing.T) {
	defer engine.InstallAmmo(engine.AmmoCalls{
		IsAmmoFull: func(client int32) bool { return client == 7 },
	})()
	if !engine.IsAmmoFull(7) || engine.IsAmmoFull(8) {
		t.Error("the installed answer was not the one called")
	}
}
