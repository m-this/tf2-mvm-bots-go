package wave

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/m-this/tf2-mvm-bots-go/internal/machine"
)

/*
Attempt is what one attempt did, written where the attempt's waves are.

The runner's own counts lived in the process: how many attempts crashed, how
many produced nothing, what the watcher complained about, what the server said
before it died. They were printed once and lost, so a session whose terminal is
gone — a context that was summarised, a shell that died, a run somebody else
started — had the files and no verdict, and reading them back reported a run
that had crashed as clean. Fifteen attempts across these transcripts produced no
wave result, and the re-runs that followed had already been played.

One of these per attempt, in the attempt's own file, next to the run record.
*/
type Attempt struct {
	Event string `json:"event"`
	Tag   string `json:"tag"`
	Arm   string `json:"arm"`
	Round int    `json:"round"`

	// Outcome is one of the four below.
	Outcome string `json:"outcome"`
	// CrashKind is the fault the container log named, empty when nothing
	// crashed. A watchdog kill, a SIGSEGV and a SIGBUS are three different
	// bugs and a run that only counted crashes could not tell them apart.
	CrashKind string `json:"crash_kind,omitempty"`
	// Refusal is what stopped the run, and Reason the watcher's complaint for
	// an attempt that played but produced nothing.
	Refusal string `json:"refusal,omitempty"`
	Reason  string `json:"reason,omitempty"`

	WavesSeen int `json:"waves_seen"`
	Samples   int `json:"samples"`

	/* The retry, which is what keeps the bed's own crashes off an arm.

	A crash with no stall and no fault behind it is replayed once on the same
	arm and the same round. Retried says it happened; Reproduced says the replay
	crashed too. Only a reproduced crash is the arm's: see mvm-9yn. */
	Retried    bool `json:"retried,omitempty"`
	Reproduced bool `json:"reproduced,omitempty"`

	StartedAt string          `json:"started_at"`
	EndedAt   string          `json:"ended_at"`
	Machine   machine.Machine `json:"machine"`
}

// The outcomes an attempt can have. They are exclusive: an attempt that crashed
// and produced no wave is a crash, because the crash is why there is no wave.
const (
	AttemptFinished = "finished"
	AttemptCrashed  = "crashed"
	AttemptEmpty    = "empty"
	AttemptRefused  = "refused"
)

// AttemptEvent is the event name the record carries.
const AttemptEvent = "attempt"

// ReadAttempt is the attempt record in a file, and whether there is one. A file
// written before these existed has none, and reads as an attempt that finished.
func ReadAttempt(path string) (Attempt, bool, error) {
	file, err := os.Open(path) //nolint:gosec // a results file the caller named
	if err != nil {
		return Attempt{}, false, err
	}
	defer func() { _ = file.Close() }()

	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		var a Attempt
		if err := json.Unmarshal(scan.Bytes(), &a); err != nil || a.Event != AttemptEvent {
			continue
		}
		return a, true, nil
	}
	return Attempt{}, false, scan.Err()
}
