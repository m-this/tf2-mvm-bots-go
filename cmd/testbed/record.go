package main

import (
	"encoding/json"
	"os"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

// writeRunRecord puts the record ahead of what the server wrote. The file is
// a few megabytes at most, so it is rewritten rather than spliced.
func writeRunRecord(path string, r wave.Run) error {
	r.Event = wave.RunEvent
	return prepend(path, r)
}

/*
writeAttemptRecord puts the runner's own verdict in the file, beside the waves
it is a verdict on.

Without it the crash count, the empty count and the watcher's reason lived in
the terminal and died with it, and -reread read a run that had crashed twice as
a clean one. See mvm-5im.
*/
func writeAttemptRecord(path string, a wave.Attempt) error {
	a.Event = wave.AttemptEvent
	return prepend(path, a)
}

func prepend(path string, record any) error {
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(path) //nolint:gosec // a results path the runner built
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// 0o600: the results are this developer's own.
	return os.WriteFile(path, append(append(line, '\n'), body...), 0o600)
}
