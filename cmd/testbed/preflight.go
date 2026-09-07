package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
	"github.com/m-this/tf2-mvm-bots-go/internal/machine"
)

/*
Refusing a machine that cannot produce a believable run, in the first second.

machine.Comparable already refuses two arms whose attempts started under the
memory floor, and it runs at the end. So a run doomed from its first second
still played every attempt, and the caller learnt after an hour that the numbers
could not be read. run.sh did not do that: mvm-ibb closed by making it read
MemAvailable and refuse to start. The guard was lost when the Go runner replaced
the shell, and this is it coming back, with the same numbers the end of the run
will judge by.

Disk is the same shape and worse. mvm-jcb was forty-seven cores at six hundred
megabytes each filling the volume, install_addons copying an extension short,
and every map load dying on Bus error, fourteen times in one container's life.
A separate session lost a build to TMPDIR being a 3.9 GB tmpfs at 99%. Neither
is visible from inside a run.
*/

/*
diskFloorMB is what the bed needs free to be trusted.

Two gibibytes: a core is six hundred megabytes, srcds_run restarts into another
one after a crash, and the entrypoint keeps two. A build staging on top of that
is what filled the volume in mvm-jcb.
*/
const diskFloorMB int64 = 2048

/*
preflight reads the machine and refuses it, before the lock is taken and before
anything is built.

The overrides are separate because the two floors are separate faults.
TESTBED_MIN_FREE_MB is memory and is the name mvm-ibb closed with;
TESTBED_MIN_DISK_MB is space. One name for both would be a name that means two
things depending on which line read it.
*/
func preflight(ctx context.Context, on server, say func(string, ...any)) error {
	played, err := machine.Snapshot("")
	if err != nil {
		return err
	}

	floorMB := overrideMB("TESTBED_MIN_FREE_MB", machine.MemAvailableFloorKB>>10)
	availableMB := played.MemAvailableKB >> 10
	if floorMB > 0 && availableMB < floorMB {
		return fmt.Errorf("%w: %d MiB of memory available and the floor is %d MiB, so the server would page and the watchdog would read a page fault as an infinite loop. TESTBED_MIN_FREE_MB=0 to play anyway",
			lab.ErrPrecondition, availableMB, floorMB)
	}
	say("%d MiB of memory available, %.2f load", availableMB, played.Load1m)

	diskMB := overrideMB("TESTBED_MIN_DISK_MB", diskFloorMB)
	for _, path := range []string{os.TempDir(), pluginBuildDir()} {
		free, err := freeMB(path)
		if err != nil {
			say("could not read the free space on %s: %v", path, err)
			continue
		}
		if diskMB > 0 && free < diskMB {
			return fmt.Errorf("%w: %s has %d MiB free and the floor is %d MiB; a short extension copy is a Bus error at every map load (mvm-jcb). TESTBED_MIN_DISK_MB=0 to play anyway",
				lab.ErrPrecondition, path, free, diskMB)
		}
		say("%s has %d MiB free", path, free)
	}

	// The container's volume is only readable through the container, so a bed
	// that is down reports nothing rather than being guessed at. It is about to
	// be recreated anyway.
	if free, cores, weightMB, ok := on.Space(ctx); ok {
		if cores > 0 {
			say("the game tree holds %d core file(s), %d MiB of them", cores, weightMB)
		}
		if diskMB > 0 && free < diskMB {
			return fmt.Errorf("%w: the game tree has %d MiB free and the floor is %d MiB, with %d core(s) in it weighing %d MiB. TESTBED_MIN_DISK_MB=0 to play anyway",
				lab.ErrPrecondition, free, diskMB, cores, weightMB)
		}
		say("the game tree has %d MiB free", free)
	}
	return nil
}

// overrideMB reads a floor out of the environment. Zero is a real value and
// means no floor; an unreadable one is the default rather than a refusal, since
// a typo in an override should not stop a run the machine can take.
func overrideMB(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func freeMB(path string) (int64, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, err
	}
	//nolint:gosec // block counts are positive and far inside int64
	return int64(fs.Bavail) * fs.Bsize >> 20, nil
}

// pluginBuildDir is where build.sh stages the tree the container bind mounts.
// It is on the host filesystem, so it is the one disk a run needs that docker
// cannot be asked about.
func pluginBuildDir() string {
	root, err := repoRoot()
	if err != nil {
		return os.TempDir()
	}
	return root
}
