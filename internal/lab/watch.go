package lab

import (
	"context"
	"fmt"
	"time"
)

/*
Health is what a poll of a running wave found, and whether it is worth going on.

A run that has gone wrong looks exactly like a slow one for the first minute and
exactly like a finished one at the end: an empty file and a timeout. The
difference is visible while it happens and nowhere afterwards, which is why this
watches rather than reading the wreckage later.
*/
type Health struct {
	Roster  Roster
	Samples int    // lines the statistics plugin has written
	Reason  string // empty while the run is worth continuing
	Fatal   bool   // the server is gone, as against the run being pointless
}

/*
Watcher decides when a run has stopped being worth waiting for.

Every rule is a real run from one session. The stall is nine Mannhunt attempts
that produced a wave with no robot in it and were read as losses. The silence is
a statistics plugin that never wrote, which reads as a mission nobody could win.
The empty RED is a wave fought by nobody, which writes a file of zeros.
*/
type Watcher struct {
	// WantDefenders is the lineup the run asked for. RED falling below it
	// mid-wave means the mod lost bots it cannot replace, and the rest of the
	// run measures a team that is not the one under test.
	WantDefenders int

	// PatienceRobots is how many polls with no robot on BLU to allow once a wave
	// has actually begun. Only then: a break has no robots on purpose, and the
	// bots spend it shopping and readying, which outlasted this patience the
	// first time it ran.
	PatienceRobots int

	// PatienceSilent is how many polls with no new sample to allow. The plugin
	// writes every five seconds, so silence across several polls is the plugin
	// gone rather than a quiet moment.
	PatienceSilent int

	// PatienceQuiet is how many rcon reads in a row may fail before the server
	// is called dead. One is not enough: a long frame makes a read time out on
	// a server that is still there, and a run once reported a crash the
	// container had never noticed.
	PatienceQuiet int

	noRobots int
	silent   int
	quiet    int
	lastSeen int
}

// Check reads whatever the server has written since the last call and reports
// what it found.
func (w *Watcher) Check(l Lab, samples int, begun bool) Health {
	roster, err := l.Roster()
	if err != nil {
		w.quiet++
		patience := w.PatienceQuiet
		if patience < 1 {
			patience = 3
		}
		if w.quiet < patience {
			// Quiet once is a long frame. Quiet repeatedly is a server that has
			// gone, and only the second is worth calling a crash.
			return Health{Samples: samples}
		}
		return Health{
			Samples: samples, Fatal: true,
			Reason: fmt.Sprintf("the server stopped answering rcon %d times running: %v", w.quiet, err),
		}
	}
	w.quiet = 0
	return w.check(roster, samples, begun)
}

// check is the rules on their own, so they can be tested without a server.
func (w *Watcher) check(roster Roster, samples int, begun bool) Health {
	h := Health{Samples: samples, Roster: roster}

	if samples > w.lastSeen {
		w.silent = 0
		w.lastSeen = samples
	} else {
		w.silent++
	}

	if roster.Robots == 0 && begun {
		w.noRobots++
	} else {
		w.noRobots = 0
	}

	switch {
	case w.PatienceSilent > 0 && w.silent >= w.PatienceSilent:
		h.Reason = fmt.Sprintf("nothing has been written down for %d polls, so the statistics plugin is not running", w.silent)
	case w.PatienceRobots > 0 && w.noRobots >= w.PatienceRobots:
		h.Reason = fmt.Sprintf("no robot has been on BLU for %d polls, so the mission is loaded and not playing", w.noRobots)
	case w.WantDefenders > 0 && roster.Defenders == 0:
		h.Reason = "RED holds no defenders, so whatever this wave measures it is not the lineup asked for"
	}
	return h
}

// Wait blocks until the condition holds or the context ends.
func (l Lab) Wait(ctx context.Context, w *Watcher, every, limit time.Duration, samples func() (int, int, bool)) (Health, error) {
	deadline := time.Now().Add(limit)
	for {
		select {
		case <-ctx.Done():
			return Health{Reason: "cancelled"}, ctx.Err()
		case <-time.After(every):
		}

		lines, waves, begun := samples()
		if waves > 0 {
			// The caller reports a count only once it has what it asked for, so
			// this is the run being finished rather than merely progressing.
			return Health{Samples: lines, Roster: Roster{}}, nil
		}
		h := w.Check(l, lines, begun)
		if h.Reason != "" {
			return h, nil
		}
		if time.Now().After(deadline) {
			return Health{Samples: lines, Reason: fmt.Sprintf("gave up after %s", limit)}, nil
		}
	}
}
