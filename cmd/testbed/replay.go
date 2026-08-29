package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

/*
Replaying a real server's settings, instead of settings the runner made up.

The stock sniper stall was reported from a player's server, chased for a day on
the test-bed, and never once reproduced. The reason was in the two server.cfg
files: he plays sm_redbots_manager_mode 2 and a lineup of
sniper,sniper,engineer,engineer,heavyweapons,soldier, and every run here was
mode 1 with three snipers and a medic. entrypoint.sh already says
"BOT_MANAGER_MODE=2 plays a mission the way a player's server does", so the gap
was known and simply never passed.

A debug bundle carries the server.cfg the player was running. Reading it back is
the difference between measuring his fault and measuring one of ours.
*/

// settingsReplayed are the convars worth carrying over. Anything else in a
// player's server.cfg is theirs: passwords, ports and hostnames belong to their
// machine and naming them here would be a way to leak one into a run.
var settingsReplayed = map[string]string{
	"sm_redbots_manager_mode":                "TESTBED_BOT_MANAGER_MODE",
	"sm_redbots_manager_team_composition":    "TESTBED_BOT_TEAM_COMP",
	"sm_redbots_manager_defender_team_size":  "TESTBED_BOT_TEAM_SIZE",
	"sm_redbots_manager_use_custom_loadouts": "TESTBED_BOT_USE_LOADOUTS",
	"sm_redbots_manager_bot_use_upgrades":    "TESTBED_BOT_USE_UPGRADES",
	"sm_redbots_engineer_nest_relocate":      "TESTBED_BOT_NEST_RELOCATE",
}

/*
readServerCfg pulls the replayable settings out of a player's server.cfg.

Returns them keyed by the compose variable that carries each, so the caller can
hand them straight to containerEnv without knowing which convar is which.
*/
func readServerCfg(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	found := map[string]string{}
	lines := bufio.NewScanner(file)

	for lines.Scan() {
		name, value, ok := settingLine(lines.Text())
		if !ok {
			continue
		}
		if key, wanted := settingsReplayed[name]; wanted {
			found[key] = value
		}
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("%s names none of the settings a run replays", path)
	}
	return found, nil
}

// settingLine splits "name value", dropping comments and the quotes around a
// value. A quoted empty value is a real setting: it is how a blacklist is cleared.
func settingLine(line string) (name, value string, ok bool) {
	if cut := strings.Index(line, "//"); cut >= 0 {
		line = line[:cut]
	}
	line = strings.TrimSpace(line)

	name, value, ok = strings.Cut(line, " ")
	if !ok {
		return "", "", false
	}
	return name, strings.Trim(strings.TrimSpace(value), `"`), true
}
