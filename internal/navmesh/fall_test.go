package navmesh

import "testing"

// TestThePluginAndTheModelAgreeOnAFall keeps one number in one place.
//
// internal/action/engineerbuildteleporter refuses a ring side whose exit has a
// hurting fall beside it, and it cannot import this package: a body may import
// internal/engine and other generated packages, and nothing else. So the
// height is written there and held to this one here, which is where the
// reasoning about sv_gravity and the landing speed lives.
func TestThePluginAndTheModelAgreeOnAFall(t *testing.T) {
	t.Parallel()

	const (
		pluginFallHurt = 264.0
		pluginRadius   = 150.0
	)

	if FallDamageHeight != pluginFallHurt {
		t.Errorf("the model says a fall hurts at %.0f and the plugin says %.0f", FallDamageHeight, pluginFallHurt)
	}
	if ExitDropRadius != pluginRadius {
		t.Errorf("the model looks %.0f round the exit and the plugin looks %.0f", ExitDropRadius, pluginRadius)
	}
}
