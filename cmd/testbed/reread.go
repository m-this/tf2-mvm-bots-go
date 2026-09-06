package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

/*
Reading a finished run again, under whatever the comparison rule is now.

A run's verdict lived in the terminal of the run that produced it and nowhere
else, so a change to the rule could not be applied to what had already been
played. mvm-k57 is the case: the fold over wave numbers was wrong, and six hours
of Coaltown had been read under it with no way to read them again.

What the files cannot give back is the runner's own count of crashes and empty
attempts, which nothing writes into them. Those read as none, and the caller is
told so rather than left to assume the run was clean.

A file written before mvm-81n carries no run record at all. Its arm comes off
its own name, which is the shape the runner has always written, and it brings no
machine with it: the comparability check then covers the attempts that did carry
one and says how many did not.
*/
// Reread is a finished run rebuilt from its files.
type Reread struct {
	Arms    []wave.Arm
	Map     string
	Mission string
	// Older is how many of the files carried no run record, so their arm came
	// off the file name and they brought no machine with them.
	Older int
}

func rereadArms(dir, tag string) (Reread, error) {
	paths, err := filepath.Glob(filepath.Join(dir, tag+"-*.jsonl"))
	if err != nil {
		return Reread{}, err
	}
	if len(paths) == 0 {
		return Reread{}, fmt.Errorf("no %s-*.jsonl under %s", tag, dir)
	}
	sort.Strings(paths)

	byName := map[string]*wave.Arm{}
	order := []string{}
	mapName, mission := "", ""
	older := 0

	for _, path := range paths {
		results, err := wave.Read(path)
		if err != nil {
			return Reread{}, err
		}

		run, found, err := wave.ReadRun(path)
		if err != nil {
			return Reread{}, err
		}
		name := run.Arm
		if !found {
			older++
			name, err = armFromName(filepath.Base(path), tag)
			if err != nil {
				return Reread{}, err
			}
		} else {
			mapName, mission = run.Map, run.Mission
		}
		// A file with no run record still says which map every wave was on.
		if mapName == "" && len(results) > 0 {
			mapName = results[0].Map
		}

		arm, seen := byName[name]
		if !seen {
			arm = &wave.Arm{Name: name, Armed: armedFeatures(run.Cvars)}
			byName[name] = arm
			order = append(order, name)
		}
		arm.Attempts++
		if found {
			arm.Machines = append(arm.Machines, run.Machine)
		}
		arm.Results = append(arm.Results, results...)
	}

	/* Which arm is the control, which the files do not say

	The runner reads the last arm as the control and gets that order off the
	command line, which is gone by the time these files are read. An arm named
	"off" is the control by habit and goes last; otherwise the name order
	decides, and printReread says which arm it landed on rather than leaving it
	to be guessed. */
	sort.SliceStable(order, func(i, j int) bool { return order[j] == "off" && order[i] != "off" })

	out := make([]wave.Arm, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return Reread{Arms: out, Map: mapName, Mission: mission, Older: older}, nil
}

// armFromName reads the arm out of tag-arm-round.jsonl, which is what a file
// written before the run record has instead of one.
func armFromName(base, tag string) (string, error) {
	rest, cut := strings.CutPrefix(strings.TrimSuffix(base, ".jsonl"), tag+"-")
	if !cut {
		return "", fmt.Errorf("%s is not named for tag %s", base, tag)
	}
	arm, _, cut := strings.Cut(rest, "-")
	if !cut || arm == "" {
		return "", fmt.Errorf("%s has no run record and no arm in its name", base)
	}
	return arm, nil
}

// printReread writes the same comparison a run prints at its end, over files.
func printReread(dir, tag string) error {
	got, err := rereadArms(dir, tag)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(report(tag, got.Map, got.Mission, got.Arms))
	fmt.Fprintf(&b, "\nRead back from the files, with %s as the control. Crashes and empty attempts are not in them and read as none.\n",
		got.Arms[len(got.Arms)-1].Name)
	if got.Older > 0 {
		fmt.Fprintf(&b, "%d file(s) carry no run record, so their arm came off the file name and they brought no machine.\n", got.Older)
	}
	_, err = fmt.Fprint(os.Stdout, b.String())
	return err
}
