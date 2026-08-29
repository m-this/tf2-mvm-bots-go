package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The settings a player runs have to survive the trip, or a run measures the
// runner's defaults and calls them the player's. See mvm-bj8.
func TestReplayCarriesThePlayersLineup(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "server.cfg")
	written := `hostname "Mann vs Archipelago"
rcon_password hunter2   // theirs, and not ours to carry
sm_redbots_manager_mode 2
sm_redbots_manager_team_composition "sniper,sniper,engineer,engineer,heavyweapons,soldier"
sm_redbots_manager_use_custom_loadouts 1
`
	if err := os.WriteFile(cfg, []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readServerCfg(cfg)
	if err != nil {
		t.Fatal(err)
	}

	for key, want := range map[string]string{
		"TESTBED_BOT_MANAGER_MODE": "2",
		"TESTBED_BOT_TEAM_COMP":    "sniper,sniper,engineer,engineer,heavyweapons,soldier",
		"TESTBED_BOT_USE_LOADOUTS": "1",
	} {
		if got[key] != want {
			t.Errorf("%s came back %q, not %q", key, got[key], want)
		}
	}

	// A password in a run's environment is a password in a log somewhere.
	for key, value := range got {
		if value == "hunter2" {
			t.Errorf("%s carried the player's rcon password over", key)
		}
	}
}

// A replayed setting has to beat the flag, or naming a config changes nothing.
func TestReplayBeatsTheFlags(t *testing.T) {
	env := containerEnv("mvm_decoy", "scout,scout", 2, map[string]string{
		"TESTBED_BOT_TEAM_COMP": "sniper,sniper",
	})

	var found string
	for _, pair := range env {
		if len(pair) > 22 && pair[:22] == "TESTBED_BOT_TEAM_COMP=" {
			found = pair[22:]
		}
	}
	if found != "sniper,sniper" {
		t.Errorf("the run plays %q, not the replayed lineup", found)
	}
}
