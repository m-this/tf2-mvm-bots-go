// Package puppet drives the test-bed's puppets: fake clients the host plugin
// seats on RED and holds a usercmd on, steered over rcon by a program instead
// of by a person.
//
// The plugin owns the body and this package owns the intent. Every verb is one
// or a few mvmbots_puppet_* commands, and the moves that need a loop, walking
// somewhere, read the status line, turn, and read it again. See mvm-n4s.
package puppet

import (
	"fmt"
	"strconv"
	"strings"
)

// Status is one puppet's line of mvmbots_puppet_status.
type Status struct {
	Index    int
	Name     string
	Class    string
	Alive    bool
	Health   int
	Healer   string
	Position Vector
	Pitch    float64
	Yaw      float64
	Currency int
	InZone   bool
}

// Vector is a point in the map, in Hammer units.
type Vector struct{ X, Y, Z float64 }

// ParseStatus reads the puppet's line out of mvmbots_puppet_status, which holds
// one line per seated puppet.
func ParseStatus(out string, index int) (Status, error) {
	prefix := "mvmbots_puppet " + strconv.Itoa(index) + " "
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return parseFields(index, strings.TrimPrefix(line, prefix))
	}
	return Status{}, fmt.Errorf("puppet %d is not in the status: %q", index, strings.TrimSpace(out))
}

func parseFields(index int, line string) (Status, error) {
	fields := map[string]string{}
	for field := range strings.FieldsSeq(line) {
		key, value, _ := strings.Cut(field, "=")
		fields[key] = value
	}
	position, ok := fields["pos"]
	if !ok {
		return Status{}, fmt.Errorf("puppet %d's status has no pos, so the host plugin predates mvmbots_puppet_input: %q", index, line)
	}
	s := Status{Index: index, Name: fields["name"], Class: fields["class"], Healer: fields["healer"]}
	var err error
	if s.Position, err = ParseVector(position); err != nil {
		return Status{}, err
	}
	for _, n := range []struct {
		key  string
		into *int
	}{{"hp", &s.Health}, {"currency", &s.Currency}} {
		if *n.into, err = strconv.Atoi(fields[n.key]); err != nil {
			return Status{}, fmt.Errorf("puppet %d's %s is %q", index, n.key, fields[n.key])
		}
	}
	for _, f := range []struct {
		key  string
		into *float64
	}{{"pitch", &s.Pitch}, {"yaw", &s.Yaw}} {
		if *f.into, err = strconv.ParseFloat(fields[f.key], 64); err != nil {
			return Status{}, fmt.Errorf("puppet %d's %s is %q", index, f.key, fields[f.key])
		}
	}
	s.Alive = fields["alive"] == "1"
	s.InZone = fields["in_zone"] == "1"
	return s, nil
}

// ParseVector reads the x,y,z the plugin prints.
func ParseVector(text string) (Vector, error) {
	parts := strings.Split(text, ",")
	if len(parts) != 3 {
		return Vector{}, fmt.Errorf("%q is not x,y,z", text)
	}
	var axes [3]float64
	for i, part := range parts {
		value, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return Vector{}, fmt.Errorf("%q is not x,y,z", text)
		}
		axes[i] = value
	}
	return Vector{axes[0], axes[1], axes[2]}, nil
}
