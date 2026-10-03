package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

/*
Refusing a server whose SourceMod is too old to buy an upgrade.

Nothing in this repository picks the SourceMod the bed runs. cm2network/tf2's
entrypoint installs it into the game volume on the container's first start, at
whatever the branch's latest was that day, and the volume then keeps it for as
long as the bed lives. A bed seeded in August is still running August's drop.

TF2 build 11076587 moved the KeyValues memory layout. SourceMod up to git7253
builds KeyValues in the old layout and reads the new one
(alliedmodders/sourcemod#2587, fixed by #2588, which is git7255), and the bots
buy their upgrades with a FakeClientCommandKeyValues MVM_Upgrade. On the old
drop every one of those is refused with nothing logged: 35 attempted and 35
refused, against 44 bought on git7255.

That is the worst shape a bed can fail in, and it is the one checkVersion
exists for too: the run completes, the file is full, and the numbers say the
bots do not buy upgrades. The reader believes it, because the bed is what they
believe. So the floor is a precondition, checked once over rcon before anything
is played, and it is named in testbed/versions.env beside the compiler pin.
*/

// SourcemodMinEnv overrides the floor for one run. Empty means no floor at all,
// which is the escape hatch for a bed somebody is deliberately running old.
const SourcemodMinEnv = "TESTBED_SOURCEMOD_MIN"

// The key the floor is written under in testbed/versions.env.
const sourcemodMinPin = "SOURCEMOD_RUNTIME_MIN"

/*
checkSourcemod refuses a run on a server below the floor.

An unreadable floor is a refusal and not a default: the pin being gone is a
thing to fix, and silently playing without a floor is how the bug it guards
against came back.
*/
func checkSourcemod(l lab.Lab, say func(string, ...any)) error {
	want, err := sourcemodFloor()
	if err != nil {
		return err
	}
	if want == "" {
		say("no SourceMod floor: %s is empty, so whatever the bed is running is accepted", SourcemodMinEnv)
		return nil
	}

	got, err := l.SourcemodVersion()
	if err != nil {
		return err
	}
	older, err := sourcemodOlder(got, want)
	if err != nil {
		return err
	}
	if older {
		return fmt.Errorf("%w: the bed is running SourceMod %s and the floor is %s. Below it a bot's MVM_Upgrade is refused with nothing logged, so the run would report the bots buying no upgrades and be believed (alliedmodders/sourcemod#2587). Recreate the bed on a fresh volume, or %s= to play anyway",
			lab.ErrPrecondition, got, want, SourcemodMinEnv)
	}
	say("the bed is running SourceMod %s, at or above the %s floor", got, want)
	return nil
}

// sourcemodFloor is the floor this run uses: the override if it is set at all,
// and the pin otherwise.
func sourcemodFloor() (string, error) {
	if value, set := os.LookupEnv(SourcemodMinEnv); set {
		return strings.TrimSpace(value), nil
	}
	pins, err := plugin.Versions()
	if err != nil {
		return "", err
	}
	want := pins[sourcemodMinPin]
	if want == "" {
		return "", fmt.Errorf("testbed/versions.env does not name %s, so there is no floor to hold the bed to", sourcemodMinPin)
	}
	return want, nil
}

// The two forms the same version is written in: 1.12.0-git7255 in the drops and
// in our pins, 1.12.0.7255 in what the server prints.
var sourcemodParts = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([0-9]+)[.-](?:git)?([0-9]+)$`)

/*
sourcemodOlder compares a version the server printed with a floor we pinned.

Four numbers each, compared in order, and both forms parse into the same four.
An unparseable version is an error rather than a pass: the comparison deciding
nothing is the one outcome that puts the old bug back.
*/
func sourcemodOlder(got, want string) (bool, error) {
	left, err := sourcemodNumbers(got)
	if err != nil {
		return false, err
	}
	right, err := sourcemodNumbers(want)
	if err != nil {
		return false, err
	}
	for i := range left {
		if left[i] != right[i] {
			return left[i] < right[i], nil
		}
	}
	return false, nil
}

func sourcemodNumbers(version string) ([4]int, error) {
	var out [4]int
	m := sourcemodParts.FindStringSubmatch(strings.TrimSpace(version))
	if m == nil {
		return out, fmt.Errorf("cannot read a SourceMod version out of %q, so the floor cannot be checked", version)
	}
	for i, field := range m[1:] {
		n, err := strconv.Atoi(field)
		if err != nil {
			return out, err
		}
		out[i] = n
	}
	return out, nil
}
