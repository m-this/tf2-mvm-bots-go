package wave

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/m-this/tf2-mvm-bots-go/internal/machine"
)

/*
Run is the first line of a results file: what the run was, so a file found a
week later says which arm, which build, which preconditions and which machine
produced it rather than leaving that to the file name.

The event name keeps it out of every reader: Read counts wave_begin and
wave_end and nothing else, and the report loaders each look for their own.
*/
type Run struct {
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
	// ReadyDelay is how long the host sat in the ready-up after each round,
	// and RelineupAfterLoss the lineup typed in the first such break: mvm-tcc.
	ReadyDelay        string `json:"ready_delay,omitempty"`
	RelineupAfterLoss string `json:"relineup_after_loss,omitempty"`
	Waves             int    `json:"waves"`
	StartWave         int    `json:"start_wave"`
	Plugin            string `json:"plugin"`
	At                string `json:"at"`
	// Injectors is the faults this arm turned on, by name. A file whose arm
	// is not written down is one nobody can read afterwards: mvm-81n.
	Injectors []string        `json:"injectors,omitempty"`
	Machine   machine.Machine `json:"machine"`
}

// RunEvent is the event name the record carries.
const RunEvent = "run"

// ReadRun is the run record at the head of a file, and whether there is one.
// A file the server wrote before the runner recorded runs has none.
func ReadRun(path string) (Run, bool, error) {
	file, err := os.Open(path) //nolint:gosec // a results file the caller named
	if err != nil {
		return Run{}, false, err
	}
	defer func() { _ = file.Close() }()

	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		var r Run
		if err := json.Unmarshal(scan.Bytes(), &r); err != nil || r.Event != RunEvent {
			continue
		}
		return r, true, nil
	}
	return Run{}, false, scan.Err()
}
