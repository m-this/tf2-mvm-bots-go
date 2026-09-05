package lab

import (
	"errors"
	"testing"
)

/*
	TestPuppetsAreNotDefenders

The whole reason the roster line names four kinds of seat. A puppet is a fake
client on RED that plays nothing, so a run counting it as a defender believes it
has the lineup it asked for while one of those seats is a statue: six bots
measured, five bots playing, and nothing in the results file able to say so.
*/
func TestPuppetsAreNotDefenders(t *testing.T) {
	for _, c := range []struct {
		name string
		line string
		want Roster
	}{
		{
			name: "the run nobody is on",
			line: "mvmbots_roster red=7 blu=22 humans=0 host=1 puppets=0",
			want: Roster{Robots: 22, Defenders: 6, Bots: 29, Host: true},
		},
		{
			name: "one puppet standing in for a player",
			line: "mvmbots_roster red=7 blu=22 humans=0 host=1 puppets=1",
			want: Roster{Robots: 22, Defenders: 5, Puppets: 1, Bots: 29, Host: true},
		},
		{
			name: "a person on the server as well",
			line: "mvmbots_roster red=8 blu=10 humans=1 host=1 puppets=1",
			want: Roster{Robots: 10, Humans: 1, Defenders: 5, Puppets: 1, Bots: 17, Host: true},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := readRoster(c.line)
			if err != nil {
				t.Fatalf("reading %q: %v", c.line, err)
			}
			if got != c.want {
				t.Errorf("read %+v, want %+v", got, c.want)
			}
		})
	}
}

/*
	TestARosterWithoutPuppetsIsRefused

The plugin and the runner ship together, so a line with no puppets field is a
server running a build from before there were any. Refusing it is the point: the
alternative is reading nought puppets off a plugin that cannot count them, which
is the same number a correct answer gives and means something else entirely.
*/
func TestARosterWithoutPuppetsIsRefused(t *testing.T) {
	_, err := readRoster("mvmbots_roster red=7 blu=22 humans=0 host=1")
	if !errors.Is(err, ErrPrecondition) {
		t.Errorf("an old roster line gave %v, want a precondition refusal", err)
	}
}
