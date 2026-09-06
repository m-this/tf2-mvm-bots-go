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

func TestReportRefusesAnArmWhoseFeatureNeverFired(t *testing.T) {
	bed := machine.Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 1, Extensions: map[string]string{}}
	quiet := []wave.Result{{Event: "wave_end", Wave: 1, Outcome: "cleared", Duration: 100, FeaturesFired: "dispenser_guard:3"}}
	loud := []wave.Result{{Event: "wave_end", Wave: 1, Outcome: "cleared", Duration: 100, FeaturesFired: "dispenser_guard:3,threat_priority:41"}}

	out := report("t", "mvm_decoy", "", []wave.Arm{
		{Name: "on", Results: quiet, Attempts: 1, Machines: []machine.Machine{bed}, Armed: []string{"threat_priority"}},
		{Name: "off", Results: quiet, Attempts: 1, Machines: []machine.Machine{bed}},
	})
	if !strings.Contains(out, "on armed threat_priority and it never fired") {
		t.Fatalf("an arm whose feature never fired was compared:\n%s", out)
	}

	out = report("t", "mvm_decoy", "", []wave.Arm{
		{Name: "on", Results: loud, Attempts: 1, Machines: []machine.Machine{bed}, Armed: []string{"threat_priority"}},
		{Name: "off", Results: quiet, Attempts: 1, Machines: []machine.Machine{bed}},
	})
	if strings.Contains(out, "never fired") {
		t.Fatalf("an arm whose feature fired was refused:\n%s", out)
	}
}

func TestArmedFeaturesComeFromTheTable(t *testing.T) {
	got := armedFeatures("sm_redbots_feature_threat_priority=1, sm_redbots_feature_spy_glance=0,sm_redbots_manager_extra=1")
	if len(got) != 1 || got[0] != "threat_priority" {
		t.Fatalf("armed %v", got)
	}
}
