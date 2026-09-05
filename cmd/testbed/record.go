package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

// writeRunRecord puts the record ahead of what the server wrote. The file is
// a few megabytes at most, so it is rewritten rather than spliced.
func writeRunRecord(path string, r wave.Run) error {
	r.Event = wave.RunEvent
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
