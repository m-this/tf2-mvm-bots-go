package puppet

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// waitMax bounds the wait verb, so a typo in a script cannot park it for a day.
const waitMax = 10 * time.Minute

// Usage is the verbs, as the program prints them.
const Usage = `verbs, one per line; n is a puppet, 1 to 4:
  seat <count>                     seat this many puppets on RED
  status                           every puppet's line
  hold <n> <buttons|none> [walk strafe]
                                   keep buttons (attack,attack2,attack3,jump,duck,use,reload)
                                   and a speed (-450 to 450) on until the next hold
  release <n>                      let go of every key
  look <n> <pitch> <yaw>           point the view and keep it there
  free <n>                         hand the view back
  slot <n> <0-5>                   switch weapon, 0 the primary
  cmd <n> <console line...>        joinclass, voicemenu 0 0, build 2 0, taunt, say ...
  tp <n> <x> <y> <z>               teleport
  walk <n> <x> <y> <z> [seconds]   run there in a straight line
  walk <n> station [seconds]       run into the nearest upgrade station
  shop <n> [itemslot row count]    open the station, buy, close it, come back
  call <n>                         press MEDIC!
  wait <seconds>                   do nothing for a while
  rcon <line...>                   any console command`

// Run does one verb and says what came of it.
func (d Driver) Run(ctx context.Context, line string) (string, error) {
	words := strings.Fields(line)
	if len(words) == 0 || strings.HasPrefix(words[0], "#") {
		return "", nil
	}
	verb, args := words[0], words[1:]
	switch verb {
	case "seat":
		if len(args) != 1 {
			return "", fmt.Errorf("seat <count>")
		}
		return d.Console.Do("mvmbots_puppet_count " + args[0])
	case "status":
		return d.Console.Do("mvmbots_puppet_status")
	case "rcon":
		return d.Console.Do(strings.Join(args, " "))
	case "wait":
		seconds, err := floatArg(args, 0, "wait <seconds>")
		if err != nil {
			return "", err
		}
		wait := time.Duration(seconds * float64(time.Second))
		if wait < 0 || wait > waitMax {
			return "", fmt.Errorf("wait is 0 to %s", waitMax)
		}
		return "", d.sleep(ctx, wait)
	}
	index, err := puppetArg(args, verb)
	if err != nil {
		return "", err
	}
	rest := args[1:]
	switch verb {
	case "hold":
		return "", d.hold(index, rest)
	case "release":
		return "", d.Release(index)
	case "look":
		pitch, err := floatArg(rest, 0, "look <n> <pitch> <yaw>")
		if err != nil {
			return "", err
		}
		yaw, err := floatArg(rest, 1, "look <n> <pitch> <yaw>")
		if err != nil {
			return "", err
		}
		return "", d.Look(index, pitch, yaw)
	case "free":
		return "", d.Free(index)
	case "slot":
		if len(rest) != 1 {
			return "", fmt.Errorf("slot <n> <0-5>")
		}
		slot, err := strconv.Atoi(rest[0])
		if err != nil || slot < 0 || slot > 5 {
			return "", fmt.Errorf("slot <n> <0-5>")
		}
		return "", d.Slot(index, slot)
	case "cmd":
		if len(rest) == 0 {
			return "", fmt.Errorf("cmd <n> <console line...>")
		}
		return "", d.Command(index, strings.Join(rest, " "))
	case "tp":
		to, err := vectorArgs(rest, "tp <n> <x> <y> <z>")
		if err != nil {
			return "", err
		}
		return "", d.Teleport(index, to)
	case "walk":
		return d.walk(ctx, index, rest)
	case "shop":
		return d.Console.Do("mvmbots_puppet_shop " + strings.Join(args, " "))
	case "call":
		return d.Console.Do("mvmbots_puppet_call " + strconv.Itoa(index))
	}
	return "", fmt.Errorf("%q is not a verb\n%s", verb, Usage)
}

func (d Driver) hold(index int, args []string) error {
	if len(args) != 1 && len(args) != 3 {
		return fmt.Errorf("hold <n> <buttons|none> [walk strafe]")
	}
	walk, strafe := 0.0, 0.0
	if len(args) == 3 {
		var err error
		if walk, err = floatArg(args, 1, "hold: walk is a number"); err != nil {
			return err
		}
		if strafe, err = floatArg(args, 2, "hold: strafe is a number"); err != nil {
			return err
		}
	}
	return d.Hold(index, args[0], walk, strafe)
}

func (d Driver) walk(ctx context.Context, index int, args []string) (string, error) {
	const usage = "walk <n> <x> <y> <z> [seconds], or walk <n> station [seconds]"
	timeout := 30 * time.Second
	var to Vector
	arrived := Near
	switch {
	case len(args) >= 1 && args[0] == "station":
		station, err := d.Station(index)
		if err != nil {
			return "", err
		}
		to, arrived, args = station, InStation, args[1:]
	case len(args) >= 3:
		var err error
		if to, err = vectorArgs(args[:3], usage); err != nil {
			return "", err
		}
		args = args[3:]
	default:
		return "", fmt.Errorf("%s", usage)
	}
	if len(args) == 1 {
		seconds, err := floatArg(args, 0, usage)
		if err != nil {
			return "", err
		}
		timeout = time.Duration(seconds * float64(time.Second))
	}
	if err := d.Walk(ctx, index, to, arrived, timeout); err != nil {
		return "", err
	}
	s, err := d.Status(index)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("arrived pos=%.0f,%.0f,%.0f in_zone=%t", s.Position.X, s.Position.Y, s.Position.Z, s.InZone), nil
}

func puppetArg(args []string, verb string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("%s needs a puppet, 1 to 4", verb)
	}
	index, err := strconv.Atoi(args[0])
	if err != nil || index < 1 || index > 4 {
		return 0, fmt.Errorf("%s: %q is not a puppet, they are 1 to 4", verb, args[0])
	}
	return index, nil
}

func floatArg(args []string, at int, usage string) (float64, error) {
	if at >= len(args) {
		return 0, fmt.Errorf("%s", usage)
	}
	value, err := strconv.ParseFloat(args[at], 64)
	if err != nil {
		return 0, fmt.Errorf("%s", usage)
	}
	return value, nil
}

func vectorArgs(args []string, usage string) (Vector, error) {
	if len(args) != 3 {
		return Vector{}, fmt.Errorf("%s", usage)
	}
	var axes [3]float64
	for i := range axes {
		value, err := floatArg(args, i, usage)
		if err != nil {
			return Vector{}, err
		}
		axes[i] = value
	}
	return Vector{axes[0], axes[1], axes[2]}, nil
}
