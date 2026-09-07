package main

import "syscall"

/*
coreDumping lets the server it starts leave a core behind.

The soft limit is nought on most machines until something raises it, and a crash
with no core is a crash nobody can look into: an unsymbolised trace kept one
watchdog crash open for two sessions, and no core at all is worse than that. It
is raised for the child alone, because a runner that raises its own limits is a
runner that changed the machine it is measuring.

Setrlimit is not in SysProcAttr, so the child is put in its own process group
and nothing else: the limit is inherited from this process, which sets it once
in main. That is the honest version and it is what the shell runner did with
ulimit -c unlimited before exec.
*/
func coreDumping() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// raiseCoreLimit puts the soft core limit up to the hard one, so a server this
// process starts can leave a core. It is best effort: a machine that refuses it
// is a machine where a native crash produces no core, which is said rather than
// fatal.
func raiseCoreLimit() error {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_CORE, &limit); err != nil {
		return err
	}
	limit.Cur = limit.Max
	return syscall.Setrlimit(syscall.RLIMIT_CORE, &limit)
}
