package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

/*
The run record: what a bed is playing, while it is playing it.

A run said what it was doing to its own terminal and to nothing else, so a
second shell asking "what is the bed on and when is it free" had ps, docker ps
and a log tail to work with, and a run somebody else started had none of those.
Every session reinvented the same watch by hand, and the answer was already
inside the runner: it polls every twenty seconds and knows the bed, the arm, the
round, the waves in, the samples and the watcher's reason.

The times are UTC. The container's clock is UTC, and a reader holding a local
timestamp against a container log line is an hour out for half the year.

The record is not the lock. The lock is a flock the kernel drops when the runner
dies; this is a file that outlives it on purpose, so the verdict of a run whose
terminal is gone can still be read. A record whose pid is gone and which never
wrote an outcome is a runner that was killed, and it says so rather than
pretending the run is still going.
*/
type State struct {
	Bed       string `json:"bed"`
	Port      string `json:"port"`
	Container string `json:"container"`
	PID       int    `json:"pid"`

	Tag     string `json:"tag"`
	Map     string `json:"map"`
	Mission string `json:"mission,omitempty"`

	// Arm and Round are the attempt being played now. Round counts from one
	// and runs to Attempts.
	Arm      string `json:"arm,omitempty"`
	Round    int    `json:"round,omitempty"`
	Attempts int    `json:"attempts"`

	WavesSeen   int `json:"waves_seen"`
	WavesWanted int `json:"waves_wanted"`
	Samples     int `json:"samples"`

	// Reason is the watcher's last complaint, empty while the run is worth
	// continuing.
	Reason string `json:"reason,omitempty"`

	StartedAt string `json:"started_at"`
	UpdatedAt string `json:"updated_at"`
	// Deadline is when the attempt in flight runs out of time.
	Deadline string `json:"deadline,omitempty"`

	// Outcome is empty while the run is going: "finished", "refused" or
	// "failed" once it is over.
	Outcome string `json:"outcome,omitempty"`
	// Error is what refused it, for a reader that has no terminal to look at.
	Error string `json:"error,omitempty"`
}

// The outcomes a finished run writes. A run still going has none of them.
const (
	outcomeFinished = "finished"
	outcomeRefused  = "refused"
	outcomeFailed   = "failed"
)

func statePath(of string) string { return filepath.Join(os.TempDir(), of+"-run.json") }

/*
Ended is whether the run that wrote this record is over.

Either it said so, or the process writing it is gone. The second case is a
runner that was killed between polls, and a reader that only trusted Outcome
would wait for it forever.
*/
func (s State) Ended() bool { return s.Outcome != "" || !alive(s.PID) }

// Stale is a record left by a runner that died without writing a verdict. The
// flock is already gone with it, so the bed is free.
func (s State) Stale() bool { return s.Outcome == "" && !alive(s.PID) }

/*
alive is whether the pid is still there.

Signal 0 is the ordinary way to ask. EPERM counts as alive: a process owned by
another user is a process, and calling somebody else's runner dead is how a bed
gets recreated out from under it.
*/
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

/*
writeState replaces the record in one step.

Written beside the target and renamed over it, so a reader polling the file
never gets half of one. The rename is what makes the read safe; a truncate and
rewrite in place is not.
*/
func writeState(s State) error {
	s.UpdatedAt = stamp(time.Now())
	body, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := statePath(s.Bed)
	// 0o600: the record names this developer's run and nothing else reads it.
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(body, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// readState is the record for a bed, and whether there is one. A bed nobody has
// run is not an error.
func readState(of string) (State, bool, error) {
	body, err := os.ReadFile(statePath(of)) //nolint:gosec // a path built from the bed name
	if errors.Is(err, os.ErrNotExist) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	var s State
	if err := json.Unmarshal(body, &s); err != nil {
		return State{}, false, fmt.Errorf("%s is not a run record: %w", statePath(of), err)
	}
	return s, true, nil
}

// stamp is how every time in the record is written: UTC, with the offset on it.
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

/*
tracker keeps the record current as the run moves.

One goroutine writes it. The runner plays one attempt at a time and the poll
callback runs on the same goroutine as the run loop, so there is no lock here
and no need for one; a second writer would be a second runner, which the bed
lock is what refuses.

A write that fails is said once and then let go. Losing the record is worth a
line of complaint and not worth ending a run that is otherwise fine.
*/
type tracker struct {
	state  State
	say    func(string, ...any)
	broken bool
}

func newTracker(s State, say func(string, ...any)) *tracker {
	s.StartedAt = stamp(time.Now())
	t := &tracker{state: s, say: say}
	t.flush()
	return t
}

func (t *tracker) update(with func(*State)) {
	if t == nil {
		return
	}
	with(&t.state)
	t.flush()
}

func (t *tracker) flush() {
	if err := writeState(t.state); err != nil && !t.broken {
		t.broken = true
		t.say("the run record could not be written, so -status and -follow will not see this run: %v", err)
	}
}

// done writes the verdict and leaves the record behind, which is the whole
// point of it: a run whose terminal is gone still has one.
func (t *tracker) done(err error) {
	t.update(func(s *State) {
		s.Arm, s.Round, s.Deadline = "", 0, ""
		switch {
		case err == nil:
			s.Outcome = outcomeFinished
		case isRefusal(err):
			s.Outcome, s.Error = outcomeRefused, err.Error()
		default:
			s.Outcome, s.Error = outcomeFailed, err.Error()
		}
	})
}
