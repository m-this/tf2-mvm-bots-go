package main

import (
	"context"
	"testing"
	"time"
)

// logSaying is a server that only has a log, which is all crash triage reads.
type logSaying string

func (l logSaying) Recreate(context.Context, string, []string) error { return nil }
func (l logSaying) Stop(context.Context) error                       { return nil }
func (l logSaying) ClearStats(context.Context) error                 { return nil }
func (l logSaying) CopyStats(context.Context, string) error          { return nil }
func (l logSaying) LogSince(context.Context, time.Time) string       { return string(l) }
func (l logSaying) NewestCore(context.Context) (string, string)      { return "", "" }
func (l logSaying) Space(context.Context) (int64, int, int64, bool)  { return 0, 0, 0, false }

/*
The install restart is not a crash, and counting it as one produced mvm-427.

srcds_run prints this line every thirty seconds for the whole life of a run that
is installing, so a substring count over an unbounded window reads dozens of
them as dozens of crashes.
*/
func TestInstallRestartIsNotACrash(t *testing.T) {
	log := logSaying(`Server restart in 10 seconds
` + restartLine + `
` + restartLine + `
` + restartLine + `
`)
	got := crashesSince(context.Background(), log, time.Now())
	if got.Fault() {
		t.Fatalf("%d restart lines read as a crash: %v", got.Restarts, got.Kinds)
	}
	if got.Restarts != 3 {
		t.Fatalf("counted %d restarts, want 3", got.Restarts)
	}
}

func TestCrashKindsAreToldApart(t *testing.T) {
	for _, c := range []struct {
		line string
		want string
	}{
		{"./srcds_run: line 344: 12 Segmentation fault      (core dumped)", crashSegv},
		{"./srcds_run: line 344: 12 Bus error               (core dumped)", crashBus},
		{"WatchDog: nonresponsive for 30 seconds, killing", crashWatchdog},
		{"Host_Error: CMapLoadHelper: Map is not valid", crashHostErr},
	} {
		got := crashesSince(context.Background(), logSaying(c.line), time.Now())
		if got.Kind() != c.want {
			t.Errorf("%q read as %s, want %s", c.line, got.Kind(), c.want)
		}
	}
}

/*
A file trodden on under the running server is the fault to name.

A run that both lost an extension mid-copy and then segfaulted has one cause and
it is the file: mvm-jcb. Naming the SIGSEGV sends the reader looking for corrupt
memory instead of a full disk.
*/
func TestTheWorstFaultIsTheOneNamed(t *testing.T) {
	log := logSaying("Segmentation fault (core dumped)\nBus error (core dumped)\n")
	if got := crashesSince(context.Background(), log, time.Now()).Kind(); got != crashBus {
		t.Fatalf("named %s, want %s", got, crashBus)
	}
}

func TestAQuietLogIsNoCrash(t *testing.T) {
	got := crashesSince(context.Background(), logSaying("L 09/07/2026 - 12:00:00: rcon from 127.0.0.1\n"), time.Now())
	if got.Fault() || got.Kind() != crashUnknown {
		t.Fatalf("an ordinary log read as a crash: %+v", got)
	}
}
