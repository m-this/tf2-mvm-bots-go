package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

The file is refused before it is opened if git would commit it. That is not
theoretical: a player's cfg was copied into plugin/testbed/ so a run could name
it, a later `git add -A testbed/` swept it in, and his rcon_password and
sv_password went to a public repository. Removing the file from the tip did not
remove it from the history. See mvm-2xs.
*/
func readServerCfg(path string) (map[string]string, error) {
	if err := refuseIfTracked(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path) //nolint:gosec // the path the caller named, checked above
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

/*
refuseIfTracked stops a replay whose cfg is somewhere git would commit it.

Asked of git rather than guessed at: a path git already ignores is fine wherever
it is, a path outside any working tree is fine, and anything else is a file one
`git add -A` away from being published. The check is the same one a person would
run, which is why it is git's answer and not a list of directory names here.
*/
func refuseIfTracked(path string) error {
	full, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)

	// No working tree above it: the file is the caller's own business.
	if !ran("git", "-C", dir, "rev-parse", "--is-inside-work-tree") {
		return nil
	}
	// Ignored, so git will not carry it however the tree is staged.
	if ran("git", "-C", dir, "check-ignore", "-q", full) {
		return nil
	}
	return fmt.Errorf("%s is inside a git working tree and is not ignored, and a server.cfg carries the player's rcon_password: put it under %s, which is ignored, or outside the tree",
		full, filepath.Join("plugin", "testbed", "replays"))
}

// ran is whether a command answered yes. git says both of the things above
// through its exit status, and the difference between "no" and "git is broken"
// does not change the answer here: either way the path is not known-ignored.
func ran(name string, args ...string) bool {
	return exec.Command(name, args...).Run() == nil //nolint:gosec // the name is a constant at every call site
}

/*
credentials never leave the cfg, and this is the assertion that says so.

settingsReplayed is a whitelist, so a password is not read in the first place;
this refuses the mistake of widening that list to something that carries one.
*/
var credentials = []string{"password", "steamaccount", "token"}

func init() {
	for convar := range settingsReplayed {
		for _, mark := range credentials {
			if strings.Contains(convar, mark) {
				panic("settingsReplayed carries " + convar + ", which is a credential: a replayed cfg must not leave the player's machine")
			}
		}
	}
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
