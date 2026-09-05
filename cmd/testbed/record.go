package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

/*
runRecord is the first line of a results file: what the run was, so a file
found a week later says which arm, which build and which preconditions
produced it rather than leaving that to the file name.

The event name keeps it out of every reader: wave.Read counts wave_begin and
wave_end and nothing else, and the report loaders each look for their own.
*/
type runRecord struct {
	Event       string `json:"event"`
	Tag         string `json:"tag"`
	Arm         string `json:"arm"`
	Cvars       string `json:"cvars"`
	Map         string `json:"map"`
	Mission     string `json:"mission"`
	Team        string `json:"team"`
	Defenders   int    `json:"defenders"`
	Puppets     int    `json:"puppets"`
	PuppetCalls bool   `json:"puppet_calls"`
	Waves       int    `json:"waves"`
	StartWave   int    `json:"start_wave"`
	Plugin      string `json:"plugin"`
	At          string `json:"at"`
}

// writeRunRecord puts the record ahead of what the server wrote. The file is
// a few megabytes at most, so it is rewritten rather than spliced.
func writeRunRecord(path string, r runRecord) error {
	r.Event = "run"
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// 0o600: the results are this developer's own.
	return os.WriteFile(path, append(append(line, '\n'), body...), 0o600)
}

/*
lastWords is what the server printed before it died, from the container's
log: the segmentation fault, the watchdog, or the Host_Error, and the lines
around it. A crash the runner can only call "rcon went quiet" is a crash
nobody can look into; run.sh used to grep the log for this and the runner
did not.
*/
func lastWords(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "docker", "logs", "--tail", "400", container()).CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	kept := make([]string, 0, 8)
	for _, line := range lines {
		if strings.Contains(line, "Segmentation fault") || strings.Contains(line, "WatchDog") ||
			strings.Contains(line, "Host_Error") || strings.Contains(line, "core dumped") ||
			strings.Contains(line, "Assertion Failed") {
			kept = append(kept, strings.TrimSpace(line))
		}
	}
	if len(kept) > 8 {
		kept = kept[len(kept)-8:]
	}
	return string(bytes.TrimSpace([]byte(strings.Join(kept, "\n"))))
}
