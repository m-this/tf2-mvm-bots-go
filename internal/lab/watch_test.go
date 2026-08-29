package lab

import (
	"strings"
	"testing"
)

// The stall that cost nine Mannhunt runs: the mission is loaded, the wave is
// running, and there is not a robot in it. That has to be named in a minute
// rather than found in an empty file after twenty five.
func TestAWaveWithNoRobotsIsCalledOut(t *testing.T) {
	w := &Watcher{WantDefenders: 6, PatienceRobots: 3, PatienceSilent: 0}
	l := Lab{}

	var h Health
	for i := 0; i < 3; i++ {
		h = w.check(Roster{Defenders: 6, Robots: 0}, 10+i, true)
	}
	_ = l

	if h.Reason == "" {
		t.Fatal("three polls with no robot said nothing")
	}
	if !strings.Contains(h.Reason, "not playing") {
		t.Errorf("the reason reads %q", h.Reason)
	}
}

// A plugin that has stopped writing looks exactly like a quiet wave, until the
// samples stop growing.
func TestSilenceIsCalledOut(t *testing.T) {
	w := &Watcher{PatienceSilent: 3, PatienceRobots: 0}

	var h Health
	for i := 0; i < 4; i++ {
		h = w.check(Roster{Defenders: 6, Robots: 12}, 100, true)
	}
	if !strings.Contains(h.Reason, "statistics plugin") {
		t.Errorf("four silent polls read as %q", h.Reason)
	}
}

// A wave with robots and a growing file is a wave worth waiting for, however
// long it takes.
func TestAHealthyWaveIsLeftAlone(t *testing.T) {
	w := &Watcher{WantDefenders: 6, PatienceRobots: 3, PatienceSilent: 3}

	for i := 0; i < 10; i++ {
		if h := w.check(Roster{Defenders: 6, Robots: 20}, 100+i*7, true); h.Reason != "" {
			t.Fatalf("a healthy wave was stopped: %q", h.Reason)
		}
	}
}

// RED emptying mid-wave means the rest of the run measures a different team.
func TestAnEmptyRedIsCalledOut(t *testing.T) {
	w := &Watcher{WantDefenders: 6, PatienceRobots: 0, PatienceSilent: 0}

	if h := w.check(Roster{Defenders: 0, Robots: 20}, 10, true); !strings.Contains(h.Reason, "no defenders") {
		t.Errorf("an empty RED read as %q", h.Reason)
	}
}

// A break has no robots on purpose, and the bots spend it shopping. Counting
// that as a stalled mission stopped a healthy run after a hundred seconds.
func TestABreakIsNotAStall(t *testing.T) {
	w := &Watcher{WantDefenders: 6, PatienceRobots: 3, PatienceSilent: 0}

	for i := 0; i < 20; i++ {
		if h := w.check(Roster{Defenders: 6, Robots: 0}, 10+i, false); h.Reason != "" {
			t.Fatalf("a break was called a stall after %d polls: %q", i+1, h.Reason)
		}
	}
}

// A single failed rcon read is a long frame, not a crash. Calling it one turned
// a healthy run into a reported crash, which is the wrong answer in the
// direction that flatters the fix being measured.
func TestOneQuietReadIsNotACrash(t *testing.T) {
	w := &Watcher{PatienceQuiet: 3}
	dead := Lab{}

	for i := 1; i <= 2; i++ {
		if h := w.Check(dead, 10, true); h.Fatal {
			t.Fatalf("read %d was called a crash", i)
		}
	}
	if h := w.Check(dead, 10, true); !h.Fatal {
		t.Error("three quiet reads running were not called a crash")
	}
}
