package stickies

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

const (
	demoClient   = int32(3)
	robotClient  = int32(5)
	launcher     = int32(102)
	bomb         = int32(200)
	noPrimary    = int32(-1)
	pipes        = int32(101)
	looseCannon  = int32(104)
	demoTeam     = engine.Team(2)
	robotTeam    = engine.Team(3)
	maxClientsSV = int32(8)
)

// world is one Demoman with one bomb on the ground and one small robot.
type world struct {
	primary     int32
	primaryAmmo bool
	bombX       float32
	robotX      float32
}

func detonates(t *testing.T, w world) bool {
	t.Helper()

	restore := []func(){
		engine.Install(engine.Calls{
			PlayerClass: func(client int32) engine.Class {
				if client == demoClient {
					return engine.ClassDemoMan()
				}
				return engine.ClassSoldier()
			},
			WeaponID: func(weapon int32) engine.Weapon {
				switch weapon {
				case pipes:
					return engine.WeaponGrenadeLauncher()
				case looseCannon:
					return engine.WeaponCannon()
				case launcher:
					return engine.WeaponPipebombLauncher()
				}
				return engine.WeaponBottle()
			},
			HasAmmo:    func(weapon int32) bool { return weapon != w.primary || w.primaryAmmo },
			EntProp:    func(int32, engine.PropType, string) int32 { return 1 },
			Origin:     func(int32) [3]float32 { return [3]float32{} },
			MaxClients: func() int32 { return maxClientsSV },
			FindEntityByClassname: func(start int32, classname string) int32 {
				if classname == "tf_projectile_pipe_remote" && start == -1 {
					return bomb
				}
				return -1
			},
			VectorDistance: func(a, b [3]float32) float32 {
				if d := b[0] - a[0]; d > 0 {
					return d
				}
				return a[0] - b[0]
			},
			IsClientInGame: func(client int32) bool { return client == demoClient || client == robotClient },
			IsPlayerAlive:  func(int32) bool { return true },
			PlayerTeam: func(client int32) engine.Team {
				if client == robotClient {
					return robotTeam
				}
				return demoTeam
			},
			WorldSpaceCenter: func(int32) [3]float32 { return [3]float32{w.robotX, 0.0, 0.0} },
			IsMiniBoss:       func(int32) bool { return false },
		}),
		engine.InstallSpy(engine.SpyCalls{
			PlayerWeaponSlot: func(_ int32, slot int32) int32 {
				if slot == engine.WeaponSlotPrimary() {
					return w.primary
				}
				return launcher
			},
		}),
		engine.InstallSappers(engine.SapperCalls{PlayerEnemyTeam: func(int32) engine.Team { return robotTeam }}),
		engine.InstallBombs(engine.BombCalls{
			OwnerEntity: func(int32) int32 { return demoClient },
			Feature:     func(int32) bool { return true },
		}),
		engine.InstallMedics(engine.MedicCalls{AbsOrigin: func(int32) [3]float32 { return [3]float32{w.bombX, 0.0, 0.0} }}),
		engine.InstallSpyChecks(engine.SpyCheckCalls{HasTheFlag: func(int32) bool { return false }}),
		engine.InstallBombInfo(engine.BombInfoCalls{IsBaseBoss: func(int32) bool { return false }}),
	}
	defer func() {
		for i := len(restore) - 1; i >= 0; i-- {
			restore[i]()
		}
	}()

	return ShouldDetonateStickies(demoClient)
}

/*
A Demoman whose stickies are the only weapon that fires, Booties and a shield or a
grenade launcher run dry, has nothing else to kill a small robot with. Holding the
button for a crowd or a giant left every lone robot walking over his bombs.
*/
func TestALoneRobotIsWorthThePressOnlyWhenStickiesAreAllHeHas(t *testing.T) {
	for _, tc := range []struct {
		name  string
		world world
		want  bool
	}{
		{"booties, lone robot on a bomb", world{primary: noPrimary, bombX: 600.0, robotX: 620.0}, true},
		{"grenade launcher run dry, lone robot on a bomb", world{primary: pipes, primaryAmmo: false, bombX: 600.0, robotX: 620.0}, true},
		{"grenade launcher loaded, lone robot on a bomb", world{primary: pipes, primaryAmmo: true, bombX: 600.0, robotX: 620.0}, false},
		{"loose cannon loaded, lone robot on a bomb", world{primary: looseCannon, primaryAmmo: true, bombX: 600.0, robotX: 620.0}, false},
		{"booties, nobody on the bomb", world{primary: noPrimary, bombX: 600.0, robotX: 1000.0}, false},
		{"booties, the bomb is under him", world{primary: noPrimary, bombX: 50.0, robotX: 60.0}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := detonates(t, tc.world); got != tc.want {
				t.Fatalf("ShouldDetonateStickies = %v, want %v", got, tc.want)
			}
		})
	}
}
