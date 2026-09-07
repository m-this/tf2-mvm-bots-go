package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
)

/*
Reading a bed without touching docker, and waiting on a run without pgrep.

These play nothing and take no lock. -status prints the record, -wait blocks on
it, and -bed list says which beds exist and who has them. Between them they
replace the watch every session wrote by hand: a log under /tmp, a sleep, a
tail, and an "until ! pgrep -f cmd/testbed" loop.
*/

// printStatus says what the bed is doing, or that nothing is.
func printStatus(of string, asJSON bool) error {
	s, found, err := readState(of)
	if err != nil {
		return err
	}
	if !found {
		if asJSON {
			return writeJSON(map[string]string{"bed": of, "state": "never run"})
		}
		fmt.Printf("%s: no run record; nothing has run this bed since it was last cleaned\n", of)
		return nil
	}
	if asJSON {
		return writeJSON(s)
	}
	fmt.Print(describe(s))
	return nil
}

/*
describe is the record in prose, and it is careful about one thing: a record
whose pid is gone is reported as ended, not as running. Reading a corpse's
fields back as live state is what sent sessions to ps and docker ps.
*/
func describe(s State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s on port %s, container %s\n", s.Bed, s.Port, s.Container)
	fmt.Fprintf(&b, "  %s: %s", s.Tag, s.Map)
	if s.Mission != "" {
		fmt.Fprintf(&b, ", %s", s.Mission)
	}
	b.WriteString("\n")

	switch {
	case s.Outcome != "":
		fmt.Fprintf(&b, "  %s at %s", s.Outcome, s.UpdatedAt)
		if s.Error != "" {
			fmt.Fprintf(&b, ": %s", s.Error)
		}
		b.WriteString("\n")
	case s.Stale():
		fmt.Fprintf(&b, "  stale: pid %d is gone and it never wrote a verdict, so the bed is free\n", s.PID)
		fmt.Fprintf(&b, "  last seen at %s\n", s.UpdatedAt)
	default:
		fmt.Fprintf(&b, "  running as pid %d, %s to %s\n", s.PID, s.StartedAt, s.UpdatedAt)
	}

	if s.Arm != "" {
		fmt.Fprintf(&b, "  playing %s, round %d of %d, %d of %d waves in, %d samples\n",
			s.Arm, s.Round, s.Attempts, s.WavesSeen, s.WavesWanted, s.Samples)
	}
	if s.Deadline != "" {
		fmt.Fprintf(&b, "  the attempt runs out of time at %s\n", s.Deadline)
	}
	if s.Reason != "" {
		fmt.Fprintf(&b, "  the watcher says: %s\n", s.Reason)
	}
	return b.String()
}

// watchEvery is how often -wait and -follow read the record. Short against the
// runner's own twenty second poll, so a transition is reported about when it
// happens rather than a poll later.
const watchEvery = 3 * time.Second

/*
waitForRun blocks until the bed's run ends, printing each transition.

It follows a bed and not a process, so it can be pointed at a run somebody else
started, and detaching from it changes nothing. It exits with the run's verdict
rather than its own: a refused run is a non-zero exit here too, which is what
makes it usable in a script that had a pgrep loop in it.
*/
func waitForRun(ctx context.Context, of string) error {
	last := ""
	for {
		s, found, err := readState(of)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("no run record for %s: nothing to wait on", of)
		}
		if line := progress(s); line != last {
			fmt.Printf("[%s] %s\n", stamp(time.Now()), line)
			last = line
		}
		if s.Ended() {
			return verdict(s)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(watchEvery):
		}
	}
}

// progress is the one line that changes when something happens, so a
// transition is a change of string rather than a diff of a struct.
func progress(s State) string {
	if s.Outcome != "" {
		return s.Outcome
	}
	if s.Stale() {
		return "stale: the runner is gone"
	}
	if s.Arm == "" {
		return "starting"
	}
	return fmt.Sprintf("%s round %d of %d, %d of %d waves, %d samples%s",
		s.Arm, s.Round, s.Attempts, s.WavesSeen, s.WavesWanted, s.Samples, said(s.Reason))
}

func said(reason string) string {
	if reason == "" {
		return ""
	}
	return " (" + reason + ")"
}

// verdict turns a finished record into this process's exit. A run that refused
// is an error here, so a caller that used to grep a log can read $?.
func verdict(s State) error {
	switch s.Outcome {
	case outcomeFinished:
		return nil
	case "":
		return fmt.Errorf("the runner (pid %d) is gone and never wrote a verdict", s.PID)
	default:
		return fmt.Errorf("the run %s: %s", s.Outcome, s.Error)
	}
}

/*
Bed is one test-bed on this machine: its compose project, its container and
whatever a runner last said about it.

A second agent picking a bed needs this. Without it the move is a raw compose
command, and compose.yml defaults its project name to the first bed, so a
compose line typed without TESTBED_PROJECT recreates somebody else's server.
*/
type Bed struct {
	Project   string `json:"project"`
	Container string `json:"container,omitempty"`
	Status    string `json:"status,omitempty"`
	Ports     string `json:"ports,omitempty"`
	Held      bool   `json:"held"`
	Playing   string `json:"playing,omitempty"`
}

/*
beds is every bed docker knows about, joined with every run record under TMPDIR.

Both halves are needed: a bed whose container was removed still has a record
worth reading, and a bed nobody has run from this checkout still has a container
that must not be recreated.
*/
func beds(ctx context.Context) ([]Bed, error) {
	found := map[string]*Bed{}

	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=com.docker.compose.service=srcds",
		"--format", "{{.Label \"com.docker.compose.project\"}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) < 4 || fields[0] == "" {
				continue
			}
			found[fields[0]] = &Bed{Project: fields[0], Container: fields[1], Status: fields[2], Ports: fields[3]}
		}
	}

	// Every record beside this bed's, by the name the runner writes them under.
	records, err := filepath.Glob(filepath.Join(os.TempDir(), "*-run.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(records)
	for _, path := range records {
		name := strings.TrimSuffix(filepath.Base(path), "-run.json")
		s, ok, err := readState(name)
		if err != nil || !ok {
			continue
		}
		bed, seen := found[name]
		if !seen {
			bed = &Bed{Project: name}
			found[name] = bed
		}
		if bed.Container == "" {
			bed.Container = s.Container
		}
		bed.Held = !s.Ended()
		bed.Playing = playing(s)
	}

	out2 := make([]Bed, 0, len(found))
	for _, bed := range found {
		out2 = append(out2, *bed)
	}
	sort.Slice(out2, func(i, j int) bool { return out2[i].Project < out2[j].Project })
	return out2, nil
}

func playing(s State) string {
	if s.Ended() {
		return fmt.Sprintf("last run %q %s at %s", s.Tag, endedAs(s), s.UpdatedAt)
	}
	return fmt.Sprintf("%q on %s, %s round %d of %d", s.Tag, s.Map, s.Arm, s.Round, s.Attempts)
}

func endedAs(s State) string {
	if s.Outcome == "" {
		return "abandoned"
	}
	return s.Outcome
}

func printBeds(ctx context.Context, asJSON bool) error {
	list, err := beds(ctx)
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(list)
	}
	if len(list) == 0 {
		fmt.Println("no bed on this machine: no srcds container and no run record")
		return nil
	}
	for _, b := range list {
		holder := "free"
		if b.Held {
			holder = "held by a runner"
		}
		fmt.Printf("%s\t%s\t%s\n", b.Project, holder, b.Status)
		if b.Ports != "" {
			fmt.Printf("  %s on %s\n", b.Container, b.Ports)
		}
		if b.Playing != "" {
			fmt.Printf("  %s\n", b.Playing)
		}
	}
	return nil
}

/*
bedAction is up, down and list, and it exists so the project name cannot be
left off.

A raw compose invocation is still possible. It stops being the obvious one,
which is the whole fix: the ten times a session found the bed busy, the next
command typed was compose up --force-recreate with no TESTBED_PROJECT, and
compose.yml defaults that to the first bed.
*/
func bedAction(ctx context.Context, action, root string, env []string, asJSON bool) error {
	compose := filepath.Join(root, "testbed", "compose.yml")
	switch action {
	case "list":
		return printBeds(ctx, asJSON)
	case "up":
		fmt.Printf("[testbed] starting %s\n", bedName())
		return lab.Compose(ctx, compose, env, "up", "-d")
	case "down":
		fmt.Printf("[testbed] stopping %s\n", bedName())
		return lab.Compose(ctx, compose, env, "stop")
	default:
		return fmt.Errorf("-bed is up, down or list, not %q", action)
	}
}

func writeJSON(v any) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}
