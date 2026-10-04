package puppet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Console is the server's rcon, as far as a puppet needs it.
type Console interface {
	Do(command string) (string, error)
}

// Walking: the top speed a player runs at, how near counts as there, how often
// the loop looks again, and how long it may go without getting closer before
// it jumps, and then gives up. A straight line with no pathing, so a wall
// between the puppet and the point is a give-up, said by name.
const (
	speedRun         = 450.0
	arriveRadius     = 40.0
	steerInterval    = 100 * time.Millisecond
	stuckJumpAfter   = 1500 * time.Millisecond
	stuckGiveUpAfter = 6 * time.Second
	walkTimeoutMax   = 2 * time.Minute
	progressMin      = 16.0
)

// ErrStuck is a walk that stopped getting closer.
var ErrStuck = errors.New("the puppet stopped getting closer")

// Driver steers one server's puppets.
type Driver struct {
	Console Console
	// Sleep waits between steering looks. Tests replace it.
	Sleep func(context.Context, time.Duration) error
}

func (d Driver) sleep(ctx context.Context, wait time.Duration) error {
	if d.Sleep != nil {
		return d.Sleep(ctx, wait)
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Status reads one puppet's line.
func (d Driver) Status(index int) (Status, error) {
	out, err := d.Console.Do("mvmbots_puppet_status")
	if err != nil {
		return Status{}, err
	}
	return ParseStatus(out, index)
}

// Hold keeps buttons and a walk speed on the puppet until the next Hold.
func (d Driver) Hold(index int, buttons string, walk, strafe float64) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_input %d %s %.0f %.0f", index, buttons, walk, strafe), "mvmbots_puppet_input puppet=")
}

// Release lets go of everything held.
func (d Driver) Release(index int) error {
	return d.Hold(index, "none", 0, 0)
}

// Look points the puppet's view and keeps it there.
func (d Driver) Look(index int, pitch, yaw float64) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_look %d %.1f %.1f", index, pitch, yaw), "mvmbots_puppet_look puppet=")
}

// Free hands the view back to the game.
func (d Driver) Free(index int) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_look %d free", index), "mvmbots_puppet_look puppet=")
}

// Command types a line into the puppet's console.
func (d Driver) Command(index int, line string) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_cmd %d %s", index, line), "mvmbots_puppet_cmd puppet=")
}

// Slot switches to the weapon in a loadout slot, 0 being the primary, the way
// a client's slot keys do: as the weapon field of its next usercmd.
func (d Driver) Slot(index, slot int) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_slot %d %d", index, slot), "mvmbots_puppet_slot puppet=")
}

// Teleport puts the puppet at a point.
func (d Driver) Teleport(index int, to Vector) error {
	return d.expect(fmt.Sprintf("mvmbots_puppet_teleport %d %.0f %.0f %.0f", index, to.X, to.Y, to.Z), "mvmbots_puppet_teleport puppet=")
}

var stationLine = regexp.MustCompile(`mvmbots_puppet_station puppet=\d+ stations=\d+ pos=(\S+)`)

// Station is where the upgrade station nearest the puppet is.
func (d Driver) Station(index int) (Vector, error) {
	out, err := d.Console.Do("mvmbots_puppet_station " + strconv.Itoa(index))
	if err != nil {
		return Vector{}, err
	}
	m := stationLine.FindStringSubmatch(out)
	if m == nil {
		return Vector{}, fmt.Errorf("no station for puppet %d: %q", index, strings.TrimSpace(out))
	}
	return ParseVector(m[1])
}

// expect runs a command and refuses a reply that is not the plugin's
// acknowledgement, because a refusal comes back as text and not as an error.
func (d Driver) expect(command, ack string) error {
	out, err := d.Console.Do(command)
	if err != nil {
		return err
	}
	if !strings.Contains(out, ack) {
		return fmt.Errorf("%s: %s", command, strings.TrimSpace(out))
	}
	return nil
}

// Arrival says when a walk is over: near the point, or, for the station, once
// the game says the puppet is in it.
type Arrival func(s Status, to Vector) bool

// Near is within arriveRadius on the floor plan.
func Near(s Status, to Vector) bool {
	return Flat(s.Position, to) <= arriveRadius
}

// InStation is the game's own word that the trigger has the puppet.
func InStation(s Status, _ Vector) bool {
	return s.InZone
}

// Walk runs the puppet at a point in a straight line, turning toward it every
// look, until arrived says so. It jumps once when it stops getting closer and
// gives up when that does not help. Whatever happens, it lets go of the keys.
func (d Driver) Walk(ctx context.Context, index int, to Vector, arrived Arrival, timeout time.Duration) (err error) {
	timeout = min(timeout, walkTimeoutMax)
	defer func() {
		if releaseErr := d.Release(index); err == nil {
			err = releaseErr
		}
	}()
	best := math.Inf(1)
	var sinceProgress time.Duration
	jumped := false
	for elapsed := time.Duration(0); elapsed < timeout; elapsed += steerInterval {
		s, err := d.Status(index)
		if err != nil {
			return err
		}
		if !s.Alive {
			return fmt.Errorf("puppet %d died on the way", index)
		}
		if arrived(s, to) {
			return nil
		}
		distance := Flat(s.Position, to)
		if distance < best-progressMin {
			best, sinceProgress, jumped = distance, 0, false
		} else {
			sinceProgress += steerInterval
		}
		if sinceProgress >= stuckGiveUpAfter {
			return fmt.Errorf("%w: %.0f units short at %.0f,%.0f,%.0f", ErrStuck, distance, s.Position.X, s.Position.Y, s.Position.Z)
		}
		buttons := "none"
		if sinceProgress >= stuckJumpAfter && !jumped {
			buttons, jumped = "jump", true
		}
		if err := d.Look(index, 0, Bearing(s.Position, to)); err != nil {
			return err
		}
		if err := d.Hold(index, buttons, speedRun, 0); err != nil {
			return err
		}
		if err := d.sleep(ctx, steerInterval); err != nil {
			return err
		}
	}
	return fmt.Errorf("puppet %d did not get there in %s", index, timeout)
}

// Bearing is the yaw, in degrees, that faces from one point to another.
func Bearing(from, to Vector) float64 {
	return math.Atan2(to.Y-from.Y, to.X-from.X) * 180 / math.Pi
}

// Flat is the distance on the floor plan, ignoring height: a trigger brush's
// centre sits in the air above where the feet go.
func Flat(a, b Vector) float64 {
	return math.Hypot(b.X-a.X, b.Y-a.Y)
}
