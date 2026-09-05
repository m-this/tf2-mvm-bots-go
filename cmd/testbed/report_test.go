package main

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/machine"
	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

func TestReportRefusesArmsFromDifferentMachines(t *testing.T) {
	bed := machine.Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 1, Extensions: map[string]string{"x.so": "a"}}
	laptop := bed
	laptop.Host = "laptop"
	result := []wave.Result{{Event: "wave_end", Wave: 1, Outcome: "cleared", Duration: 100}}

	out := report("t", "mvm_decoy", "", []wave.Arm{
		{Name: "on", Results: result, Attempts: 1, Machines: []machine.Machine{bed}},
		{Name: "off", Results: result, Attempts: 1, Machines: []machine.Machine{laptop}},
	})
	if !strings.Contains(out, "on against off is not compared") || !strings.Contains(out, "bed and laptop") {
		t.Fatalf("the report compared arms from two machines:\n%s", out)
	}

	out = report("t", "mvm_decoy", "", []wave.Arm{
		{Name: "on", Results: result, Attempts: 1, Machines: []machine.Machine{bed}},
		{Name: "off", Results: result, Attempts: 1, Machines: []machine.Machine{bed}},
	})
	if strings.Contains(out, "not compared") {
		t.Fatalf("the same machine was refused:\n%s", out)
	}
}
