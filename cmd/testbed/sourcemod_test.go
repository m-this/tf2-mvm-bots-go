package main

import "testing"

/*
The two forms of a SourceMod version compare, and the drift that mattered is the
one it catches.

git7253 against a git7255 floor is the whole reason this exists: that is the
drop whose KeyValues layout silently refuses every upgrade the bots buy, and the
server prints its version as 1.12.0.7253 while the pin is written 1.12.0-git7253.
A comparison that cannot see those two as the same number is a floor that never
fires.
*/
func TestASourcemodVersionComparesAgainstTheFloor(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		got, want string
		older     bool
	}{
		{"1.12.0.7253", "1.12.0-git7255", true},
		{"1.12.0.7255", "1.12.0-git7255", false},
		{"1.12.0.7256", "1.12.0-git7255", false},
		{"1.12.0-git7255", "1.12.0-git7255", false},
		{"1.11.0.6968", "1.12.0-git7255", true},
		{"1.13.0.100", "1.12.0-git7255", false},
		// The bed's own compiler pin, which is not a server to play on.
		{"1.12.0.7164", "1.12.0-git7255", true},
	} {
		older, err := sourcemodOlder(c.got, c.want)
		if err != nil {
			t.Errorf("%s against %s: %v", c.got, c.want, err)
			continue
		}
		if older != c.older {
			t.Errorf("%s older than %s = %v, want %v", c.got, c.want, older, c.older)
		}
	}
}

// A version nobody can read is an error and never a pass. A floor that decides
// nothing is the same as no floor, and no floor is what let git7253 through.
func TestAnUnreadableSourcemodVersionIsAnError(t *testing.T) {
	t.Parallel()

	for _, got := range []string{"", "1.12", "1.12.0", "unknown", "1.12.0-git"} {
		if _, err := sourcemodOlder(got, "1.12.0-git7255"); err == nil {
			t.Errorf("%q compared without complaint, want an error", got)
		}
	}
}
