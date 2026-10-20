package botqueries

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

const (
	demoClient   = int32(3)
	demoPipes    = int32(101)
	demoStickies = int32(102)
	demoBottle   = int32(103)
	demoThreat   = engine.Known(7)
	demoRobot    = int32(40)
)

// demoLoadout is one Demoman's two launchers, and what the rules are asked.
type demoLoadout struct {
	pipesClip, pipesReserve       int32
	stickiesClip, stickiesReserve int32
	threat                        engine.Known
	stickiesWanted                bool
}

// equip runs the weapon choice over the loadout and says what he ended up
// holding and whether reload was pressed.
func equip(t *testing.T, l demoLoadout) (held int32, reloaded bool) {
	t.Helper()

	held = -1
	restore := []func(){
		engine.Install(engine.Calls{
			PlayerClass:    func(int32) engine.Class { return engine.ClassDemoMan() },
			Origin:         func(int32) [3]float32 { return [3]float32{} },
			VectorDistance: func(a, b [3]float32) float32 { return b[0] - a[0] },
			IsPlayer:       func(int32) bool { return true },
			IsInvulnerable: func(int32) bool { return false },
			WeaponID: func(weapon int32) engine.Weapon {
				switch weapon {
				case demoPipes:
					return engine.WeaponGrenadeLauncher()
				case demoStickies:
					return engine.WeaponPipebombLauncher()
				}
				return engine.WeaponBottle()
			},
			HasAmmo: func(weapon int32) bool {
				switch weapon {
				case demoPipes:
					return l.pipesClip+l.pipesReserve > 0
				case demoStickies:
					return l.stickiesClip+l.stickiesReserve > 0
				}
				return true
			},
		}),
		engine.InstallSpy(engine.SpyCalls{
			IsValidEntity: func(entity int32) bool { return entity > 0 },
			PlayerWeaponSlot: func(_ int32, slot int32) int32 {
				switch slot {
				case engine.WeaponSlotPrimary():
					return demoPipes
				case engine.WeaponSlotSecondary():
					return demoStickies
				}
				return demoBottle
			},
			SetPlayerActiveWeapon: func(_ int32, weapon int32) { held = weapon },
		}),
		engine.InstallSappers(engine.SapperCalls{
			AmmoCount: func(_ int32, ammo int32) int32 {
				if ammo == engine.AmmoPrimary() {
					return l.pipesReserve
				}
				return l.stickiesReserve
			},
		}),
		engine.InstallQueries(engine.QueryCalls{
			WasEverVisible:       func(engine.Known) bool { return true },
			TimeSinceLastSeen:    func(engine.Known) float32 { return 0.0 },
			LastKnownPosition:    func(engine.Known) [3]float32 { return [3]float32{600.0, 0.0, 0.0} },
			StickyLauncherWanted: func(int32, int32, int32, float32) bool { return l.stickiesWanted },
			Clip1Of: func(weapon int32) int32 {
				if weapon == demoPipes {
					return l.pipesClip
				}
				return l.stickiesClip
			},
			WeaponMaxClip: func(weapon int32) int32 {
				if weapon == demoPipes {
					return 4
				}
				return 8
			},
			PressButtonsNow: func(_ engine.Buttons, buttons int32) {
				reloaded = reloaded || buttons&engine.InReload() != 0
			},
		}),
		engine.InstallThreats(engine.ThreatCalls{
			Entity:        func(engine.Known) int32 { return demoRobot },
			IsMeleeWeapon: func(weapon int32) bool { return weapon == demoBottle },
		}),
		engine.InstallBots(engine.BotCalls{NextBotOf: func(int32) engine.Bot { return 1 }}),
		engine.InstallLurks(engine.LurkCalls{ExtraButtons: func(int32) engine.Buttons { return 1 }}),
	}
	defer func() {
		for i := len(restore) - 1; i >= 0; i-- {
			restore[i]()
		}
	}()

	EquipBestWeaponForThreat(demoClient, l.threat)

	return held, reloaded
}

/*
Empty pipes against a robot the sticky rule turned down is a reload, not eight
bombs nobody presses for. ShouldDetonateStickies wants a crowd, a giant, the
carrier or a Medic, so a lone robot never sets them off: the clip went for
nothing and left the launcher empty.
*/
func TestEmptyPipesReloadRatherThanFireStickiesNobodyDetonates(t *testing.T) {
	held, _ := equip(t, demoLoadout{
		pipesClip: 0, pipesReserve: 16,
		stickiesClip: 8, stickiesReserve: 24,
		threat: demoThreat,
	})
	if held != demoPipes {
		t.Fatalf("held %d, want the pipes (%d) to reload", held, demoPipes)
	}
}

// The case the sticky rule exists for still gets the launcher.
func TestAWantedTargetGetsTheLoadedLauncher(t *testing.T) {
	held, _ := equip(t, demoLoadout{
		pipesClip: 4, pipesReserve: 16,
		stickiesClip: 8, stickiesReserve: 24,
		threat: demoThreat, stickiesWanted: true,
	})
	if held != demoStickies {
		t.Fatalf("held %d, want the stickies (%d)", held, demoStickies)
	}
}

// Pipes with nothing left at all is still the launcher, whatever the target.
func TestPipesWithNoAmmoLeftFallBackToTheLauncher(t *testing.T) {
	held, _ := equip(t, demoLoadout{
		pipesClip: 0, pipesReserve: 0,
		stickiesClip: 8, stickiesReserve: 24,
		threat: demoThreat,
	})
	if held != demoStickies {
		t.Fatalf("held %d, want the stickies (%d)", held, demoStickies)
	}
}

/*
The game reloads only the weapon in hand, and every rule that hands him the
launcher wants a loaded clip. A clip spent on a giant stayed spent until he died,
and he met the next giant with pipes. Between fights is when it gets reloaded.
*/
func TestASpentLauncherIsReloadedBetweenFights(t *testing.T) {
	held, reloaded := equip(t, demoLoadout{
		pipesClip: 4, pipesReserve: 16,
		stickiesClip: 0, stickiesReserve: 24,
		threat: engine.NoKnownEntity(),
	})
	if held != demoStickies || !reloaded {
		t.Fatalf("held %d, reload pressed %v; want the stickies (%d) in hand and reload pressed", held, reloaded, demoStickies)
	}
}

// A full launcher, or one with nothing left to load, is not a reason to put the pipes away.
func TestNothingToReloadKeepsThePipes(t *testing.T) {
	for _, l := range []demoLoadout{
		{pipesClip: 4, pipesReserve: 16, stickiesClip: 8, stickiesReserve: 24},
		{pipesClip: 4, pipesReserve: 16, stickiesClip: 2, stickiesReserve: 0},
	} {
		l.threat = engine.NoKnownEntity()
		held, reloaded := equip(t, l)
		if held != demoPipes || reloaded {
			t.Fatalf("%+v: held %d, reload pressed %v; want the pipes (%d) and no reload", l, held, reloaded, demoPipes)
		}
	}
}
