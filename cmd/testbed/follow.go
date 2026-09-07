package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

/*
Watching wave results land while the run is still playing.

Nothing printed a wave result until its attempt was over, which on a six-wave
Mannhattan is forty minutes after the first wave landed, so every session that
wanted to know how it was going read the raw file by hand: a tail of the log, a
docker exec wc -l over the statistics file, a Python script counting outcomes.

The data was already local and already fresh. The runner copies the statistics
file out of the container at every poll, into TMPDIR, because that is how it
watches the wave itself. This reads the same file.

The argument for it is the same one internal/lab.Watcher is built on: a wave
that goes wrong looks like a slow one for its first minute. A run whose first
two waves already say what the change did is a run that can be stopped at
attempt two instead of attempt six.
*/
func follow(ctx context.Context, of string, asJSON bool) error {
	staged := filepath.Join(os.TempDir(), of+"-stats.jsonl")
	tally := map[string][]wave.Result{}
	var order []string

	// The statistics file is cleared at the start of every attempt, so what is
	// in it belongs to the attempt named in the record. Waves already reported
	// for that attempt are counted, not remembered: the file only grows within
	// an attempt.
	attempt, reported := "", 0
	said := ""

	for {
		s, found, err := readState(of)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("no run record for %s: there is nothing to follow", of)
		}
		if !asJSON && said == "" {
			fmt.Print(heading(s))
			said = "yes"
		}

		if now := s.Arm + "/" + fmt.Sprint(s.Round); now != attempt {
			attempt, reported = now, 0
		}

		results, err := wave.Read(staged)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, r := range results[min(reported, len(results)):] {
			if !contains(order, s.Arm) {
				order = append(order, s.Arm)
			}
			tally[s.Arm] = append(tally[s.Arm], r)
			if err := sayWave(s, r, asJSON); err != nil {
				return err
			}
		}
		reported = len(results)

		if s.Ended() {
			if !asJSON {
				fmt.Print(standings(order, tally))
			}
			return verdict(s)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(watchEvery):
		}
	}
}

func heading(s State) string {
	line := fmt.Sprintf("following %s: %q on %s", s.Bed, s.Tag, s.Map)
	if s.Mission != "" {
		line += ", " + s.Mission
	}
	return line + fmt.Sprintf(", %d attempts of %d waves\n", s.Attempts, s.WavesWanted)
}

// sayWave is one line per wave as it lands, with the arm and the round it
// belongs to, because a follower that attached mid-run has no other way to know.
func sayWave(s State, r wave.Result, asJSON bool) error {
	if asJSON {
		return writeJSONLine(map[string]any{
			"at": stamp(time.Now()), "tag": s.Tag, "arm": s.Arm, "round": s.Round,
			"wave": r.Wave, "outcome": r.Outcome, "robot_kills": r.RobotKills,
			"defender_deaths": r.Deaths, "damage": r.Damage, "features_fired": r.FeaturesFired,
		})
	}
	_, err := fmt.Printf("  %-4s round %d  wave %d  %-8s  %d robots killed, %d defenders dead, %d damage\n",
		s.Arm, s.Round, r.Wave, r.Outcome, r.RobotKills, r.Deaths, r.Damage)
	return err
}

// standings is the shape of the comparison so far, which is what makes the
// follow worth reading before the run is over.
func standings(order []string, tally map[string][]wave.Result) string {
	out := "\n"
	sorted := append([]string(nil), order...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[j] == "off" && sorted[i] != "off" })
	for _, arm := range sorted {
		got := tally[arm]
		out += fmt.Sprintf("%s: %d waves, %d cleared, %d robots killed, %d defenders dead\n",
			arm, len(got), cleared(got), sum(got, func(r wave.Result) int { return r.RobotKills }),
			sum(got, func(r wave.Result) int { return r.Deaths }))
	}
	return out
}

// writeJSONLine is one object per line, for a reader that is not a person.
func writeJSONLine(v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}

func sum(of []wave.Result, pick func(wave.Result) int) int {
	n := 0
	for _, r := range of {
		n += pick(r)
	}
	return n
}

func contains(list []string, want string) bool {
	for _, got := range list {
		if got == want {
			return true
		}
	}
	return false
}
