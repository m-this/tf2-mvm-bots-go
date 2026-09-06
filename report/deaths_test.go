package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeathsAndUpgradesAreReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.jsonl")
	body := `{"event":"defender_death","map":"mvm_decoy","wave":1,"at":12.5,"who":"Scout","class":"scout","killer":"heavy","giant":true,"weapon":"minigun","cause":"bullet"}
{"event":"upgrade","map":"mvm_decoy","wave":1,"who":"Scout","class":"scout","slot":-1,"upgrade":4,"count":1}
{"event":"upgrade","map":"mvm_decoy","wave":2,"who":"Scout","class":"scout","slot":-1,"upgrade":4,"count":2}
{"event":"wave_end","wave":1}
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	deaths, upgrades, err := loadDeathsAndUpgrades(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(deaths) != 1 || deaths[0].Killer != "heavy" || !deaths[0].Giant || deaths[0].Cause != "bullet" {
		t.Errorf("deaths %+v", deaths)
	}
	if len(upgrades) != 2 || upgrades[1].Count != 2 {
		t.Errorf("upgrades %+v", upgrades)
	}
}
