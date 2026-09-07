package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

/*
What killed the server, in a window that belongs to this attempt.

Every session that saw a crash typed the same line, a hundred and seventeen
times across the transcripts on this machine:

	docker logs <bed> 2>&1 | grep -icE "core dumped|Segmentation fault|Bus error"

which is wrong twice over. docker logs with no --since keeps the container's
whole life across restarts, so a count taken over that window charges this
attempt with crashes from hours ago; mvm-427 was filed on exactly that inference
and closed as one. And a bare count says nothing about which of three unrelated
faults happened.

run.sh named the kind and printed the command that turns the newest core into a
stack, which is mvm-b3j. run.sh is gone and the Go runner did not inherit it.
This is that, in Go, scoped to the attempt.
*/

// Crash kinds. They are not variations on one fault: a watchdog kill means
// something was slow, a SIGSEGV means something is corrupt, a SIGBUS means a
// file was rewritten under the running server (mvm-jcb).
const (
	crashWatchdog = "watchdog"
	crashSegv     = "SIGSEGV"
	crashBus      = "SIGBUS"
	crashHostErr  = "Host_Error"
	crashUnknown  = "unknown"
)

// Crash is what the log said, over a window the caller can see.
type Crash struct {
	// Since is the start of the window read, so the reader can tell a fresh
	// crash from one the container remembers.
	Since string `json:"since"`
	// Kinds is each fault found and how many times, by the names above.
	Kinds map[string]int `json:"kinds,omitempty"`
	// Restarts is srcds_run putting the server back after it died. Not a
	// crash, and counting it as one is what produced mvm-427.
	Restarts int `json:"restarts"`
	// Lines is the log lines the kinds were read off, newest last.
	Lines []string `json:"lines,omitempty"`
	// Core is the newest core file in the container, and Symbolise the command
	// that turns it into a backtrace.
	Core      string `json:"core,omitempty"`
	Symbolise string `json:"symbolise,omitempty"`
}

// Kind is the one fault to name, worst first. A run that both trod on a file
// and segfaulted has one cause and it is the file.
func (c Crash) Kind() string {
	for _, kind := range []string{crashBus, crashSegv, crashWatchdog, crashHostErr} {
		if c.Kinds[kind] > 0 {
			return kind
		}
	}
	return crashUnknown
}

// Fault is whether anything worth calling a crash was found. A window with
// nothing but install restarts in it is not one.
func (c Crash) Fault() bool { return len(c.Kinds) > 0 }

/*
The lines each kind is read off.

The install restart is the trap. srcds_run prints it every thirty seconds for
the whole life of a run that is installing, so a substring match over an
unbounded window counts dozens of them, and the count reads as dozens of
crashes. It is matched here so it can be excluded by name rather than by luck.
*/
var crashLines = []struct {
	kind  string
	marks []string
}{
	{crashBus, []string{"Bus error"}},
	{crashSegv, []string{"Segmentation fault", "SIGSEGV"}},
	{crashWatchdog, []string{"WatchDog", "Watchdog", "engine watchdog"}},
	{crashHostErr, []string{"Host_Error", "Assertion Failed"}},
}

const restartLine = "Add \"-debug\" to the ./srcds_run command line"

/*
crashesSince reads the container log from a point in time and classifies it.

The window is the attempt's own start, in UTC, because the container's clock is
UTC. docker takes an RFC3339 stamp for --since, which is exact; the relative
form ("20m") is a moving target between one poll and the next.
*/
func crashesSince(ctx context.Context, on server, since time.Time) Crash {
	c := Crash{Since: stamp(since), Kinds: map[string]int{}}

	out := on.LogSince(ctx, since)
	if out == "" {
		return c
	}

	kept := make([]string, 0, 8)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.Contains(line, restartLine) {
			c.Restarts++
			continue
		}
		for _, want := range crashLines {
			if !anyOf(line, want.marks) {
				continue
			}
			c.Kinds[want.kind]++
			kept = append(kept, strings.TrimSpace(line))
			break
		}
	}
	if len(kept) > 8 {
		kept = kept[len(kept)-8:]
	}
	c.Lines = kept

	if c.Fault() {
		c.Core, c.Symbolise = on.NewestCore(ctx)
	}
	return c
}

func anyOf(line string, marks []string) bool {
	for _, mark := range marks {
		if strings.Contains(line, mark) {
			return true
		}
	}
	return false
}

// say writes the crash the way a reader wants it: the kind, the window it was
// read over, and the one command that turns it into a backtrace.
func (c Crash) say(to func(string, ...any)) {
	if !c.Fault() {
		if c.Restarts > 0 {
			to("no crash in the log since %s; %d srcds_run restart lines, which are not crashes", c.Since, c.Restarts)
		}
		return
	}
	to("crash: %s, from the log since %s", c.Kind(), c.Since)
	for _, kind := range sortedKinds(c.Kinds) {
		to("  %s x%d", kind, c.Kinds[kind])
	}
	for _, line := range c.Lines {
		to("  %s", line)
	}
	if c.Core != "" {
		to("  the newest core is %s; for a backtrace:", c.Core)
		to("    %s", c.Symbolise)
	}
}

func sortedKinds(of map[string]int) []string {
	out := make([]string, 0, len(of))
	for kind := range of {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}

// printCrashes is -crashes: triage after the fact, without docker in the
// caller's hands. The window defaults to something a person would have typed
// and can be given as any duration.
func printCrashes(ctx context.Context, on server, window time.Duration, asJSON bool) error {
	c := crashesSince(ctx, on, time.Now().Add(-window))
	if asJSON {
		return writeJSON(c)
	}
	fmt.Printf("%s, the last %s\n", container(), window)
	if !c.Fault() {
		fmt.Printf("  no crash; %d srcds_run restart lines, which are not crashes\n", c.Restarts)
		return nil
	}
	c.say(func(format string, args ...any) { fmt.Printf(format+"\n", args...) })
	return nil
}
