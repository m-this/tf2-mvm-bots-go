package puppet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

const statusOut = `mvmbots_puppet 1 name=testbed-player-1 class=scout alive=1 hp=125 healer= pos=-120,340,8 pitch=0 yaw=90 currency=400 in_zone=0 buttons=0 forward=0 side=0
mvmbots_puppet 2 name=testbed-player-2 class=heavy alive=0 hp=0 healer=Medic pos=10,20,30 pitch=-5 yaw=180 currency=0 in_zone=1 buttons=1 forward=450 side=0`

func TestParseStatusReadsThePuppetAskedFor(t *testing.T) {
	s, err := ParseStatus(statusOut, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := Status{
		Index: 2, Name: "testbed-player-2", Class: "heavy", Healer: "Medic",
		Position: Vector{10, 20, 30}, Pitch: -5, Yaw: 180, InZone: true,
	}
	if s != want {
		t.Errorf("got %+v\nwant %+v", s, want)
	}
	first, err := ParseStatus(statusOut, 1)
	if err != nil || !first.Alive || first.Health != 125 || first.Currency != 400 || first.Healer != "" {
		t.Errorf("puppet 1 = %+v, %v", first, err)
	}
}

// The plugin before mvmbots_puppet_input printed no position. A walk on it
// would steer on zeros, so it is refused by name.
func TestParseStatusRefusesAnOldPlugin(t *testing.T) {
	_, err := ParseStatus("mvmbots_puppet 1 name=p class=scout alive=1 hp=125 healer=", 1)
	if err == nil || !strings.Contains(err.Error(), "predates") {
		t.Errorf("err = %v", err)
	}
	if _, err := ParseStatus(statusOut, 3); err == nil {
		t.Error("a puppet that is not seated parsed")
	}
}

func TestBearingFacesThePoint(t *testing.T) {
	for _, c := range []struct {
		to   Vector
		want float64
	}{{Vector{100, 0, 0}, 0}, {Vector{0, 100, 0}, 90}, {Vector{-100, 0, 0}, 180}, {Vector{0, -100, 50}, -90}} {
		if got := Bearing(Vector{}, c.to); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Bearing to %v = %v, want %v", c.to, got, c.want)
		}
	}
}

// fakeServer moves the puppet a fixed step toward wherever it was last told
// to look, while forward is held, the way the game does with a usercmd.
type fakeServer struct {
	pos      Vector
	yaw      float64
	walking  bool
	step     float64
	wall     float64 // x the puppet cannot pass, 0 for none
	commands []string
}

func (f *fakeServer) Do(command string) (string, error) {
	f.commands = append(f.commands, command)
	words := strings.Fields(command)
	switch words[0] {
	case "mvmbots_puppet_status":
		if f.walking {
			rad := f.yaw * math.Pi / 180
			f.pos.X += f.step * math.Cos(rad)
			f.pos.Y += f.step * math.Sin(rad)
			if f.wall != 0 && f.pos.X > f.wall {
				f.pos.X = f.wall
			}
		}
		return fmt.Sprintf("mvmbots_puppet 1 name=p class=scout alive=1 hp=125 healer= pos=%.0f,%.0f,%.0f pitch=0 yaw=%.0f currency=400 in_zone=0 buttons=0 forward=0 side=0",
			f.pos.X, f.pos.Y, f.pos.Z, f.yaw), nil
	case "mvmbots_puppet_look":
		_, _ = fmt.Sscanf(words[3], "%g", &f.yaw)
		return "mvmbots_puppet_look puppet=1", nil
	case "mvmbots_puppet_input":
		f.walking = words[3] != "0"
		return "mvmbots_puppet_input puppet=1", nil
	}
	return "", fmt.Errorf("unexpected %q", command)
}

func noSleep(context.Context, time.Duration) error { return nil }

func TestWalkGetsThereAndLetsGo(t *testing.T) {
	server := &fakeServer{step: 30}
	d := Driver{Console: server, Sleep: noSleep}
	if err := d.Walk(context.Background(), 1, Vector{300, 400, 0}, Near, time.Minute); err != nil {
		t.Fatal(err)
	}
	if Flat(server.pos, Vector{300, 400, 0}) > arriveRadius {
		t.Errorf("stopped at %v", server.pos)
	}
	if last := server.commands[len(server.commands)-1]; last != "mvmbots_puppet_input 1 none 0 0" {
		t.Errorf("the walk ended holding keys: last command %q", last)
	}
}

// A wall is a give-up said by name, after one jump, and the keys are let go.
func TestWalkGivesUpAtAWall(t *testing.T) {
	server := &fakeServer{step: 30, wall: 100}
	d := Driver{Console: server, Sleep: noSleep}
	err := d.Walk(context.Background(), 1, Vector{500, 0, 0}, Near, time.Minute)
	if !errors.Is(err, ErrStuck) {
		t.Fatalf("err = %v, want ErrStuck", err)
	}
	jumps := 0
	for _, c := range server.commands {
		if strings.Contains(c, " jump ") {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("jumped %d times, want once", jumps)
	}
	if last := server.commands[len(server.commands)-1]; last != "mvmbots_puppet_input 1 none 0 0" {
		t.Errorf("the walk ended holding keys: last command %q", last)
	}
}

// A refusal from the plugin comes back as text, and is an error here.
func TestARefusalIsAnError(t *testing.T) {
	d := Driver{Console: consoleFunc(func(string) (string, error) {
		return "mvmbots_puppet_input: puppet 1 is not seated", nil
	})}
	if err := d.Release(1); err == nil || !strings.Contains(err.Error(), "not seated") {
		t.Errorf("err = %v", err)
	}
}

type consoleFunc func(string) (string, error)

func (f consoleFunc) Do(command string) (string, error) { return f(command) }

func TestRunTranslatesVerbs(t *testing.T) {
	var sent []string
	d := Driver{Console: consoleFunc(func(c string) (string, error) {
		sent = append(sent, c)
		return strings.Fields(c)[0] + " puppet=1", nil
	}), Sleep: noSleep}
	for _, line := range []string{
		"hold 1 attack,jump 450 -200",
		"look 1 -10 135",
		"cmd 1 joinclass heavyweapons",
		"tp 1 1 2 3",
		"slot 1 2",
		"free 1",
		"# a comment",
		"",
		"wait 1",
	} {
		if _, err := d.Run(context.Background(), line); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
	}
	want := []string{
		"mvmbots_puppet_input 1 attack,jump 450 -200",
		"mvmbots_puppet_look 1 -10.0 135.0",
		"mvmbots_puppet_cmd 1 joinclass heavyweapons",
		"mvmbots_puppet_teleport 1 1 2 3",
		"mvmbots_puppet_slot 1 2",
		"mvmbots_puppet_look 1 free",
	}
	if strings.Join(sent, "\n") != strings.Join(want, "\n") {
		t.Errorf("sent\n%s\nwant\n%s", strings.Join(sent, "\n"), strings.Join(want, "\n"))
	}
	for _, bad := range []string{"hold 5 none", "look 1 x 0", "slot 1 6", "wait 99999", "fly 1", "walk 1"} {
		if _, err := d.Run(context.Background(), bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
