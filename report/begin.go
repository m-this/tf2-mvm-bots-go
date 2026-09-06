package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// waveBegin is the line the statistics plugin writes as a wave starts. Only
// what a report reads is named here.
type waveBegin struct {
	Wave int `json:"wave"`
	// MedicCharge is the highest uber charge on a RED medigun as the wave
	// begins, and -1 when RED had no medic holding one.
	MedicCharge float64 `json:"medic_charge"`
}

func loadWaveBegins(path string) ([]waveBegin, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var begins []waveBegin
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, `{"event":"wave_begin"`) || !strings.Contains(line, `"medic_charge"`) {
			continue
		}
		b := waveBegin{MedicCharge: -1}
		if json.Unmarshal([]byte(line), &b) == nil && b.Wave > 0 {
			begins = append(begins, b)
		}
	}
	return begins, scanner.Err()
}

// printUberAtWaveStart says what the medics brought into each wave. A charge
// under one at the start of any wave after the first is a break spent not
// healing, which is mvm-bk8.
func printUberAtWaveStart(begins []waveBegin) {
	if len(begins) == 0 {
		return
	}
	fmt.Printf("\n  uber charge at wave start\n")
	for _, b := range begins {
		switch {
		case b.MedicCharge < 0:
			fmt.Printf("    wave %2d  no medic holding a medigun\n", b.Wave)
		default:
			fmt.Printf("    wave %2d  %3.0f%%\n", b.Wave, b.MedicCharge*100)
		}
	}
}
