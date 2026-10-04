// Command puppet drives the test-bed's puppets over rcon: one verb from the
// command line, or a stream of them on stdin, one per line, so a harness can
// steer a puppet as it goes. TESTBED_PORT picks the bed, as it does for rc.
//
//	go run ./cmd/puppet walk 1 station
//	printf 'seat 1\nwait 12\nwalk 1 station\nshop 1 -1 59 1\n' | go run ./cmd/puppet -
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/m-this/tf2-mvm-bots-go/internal/puppet"
	"github.com/m-this/tf2-mvm-bots-go/internal/rcon"
)

// lineMax bounds one line of a script, which is a verb and its numbers.
const lineMax = 4096

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "puppet:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("a verb, or - to read verbs from stdin\n%s", puppet.Usage)
	}
	port := os.Getenv("TESTBED_PORT")
	if port == "" {
		port = "27025"
	}
	password := os.Getenv("TESTBED_RCONPW")
	if password == "" {
		password = "testbed"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	d := puppet.Driver{Console: rcon.Client{Addr: "127.0.0.1:" + port, Password: password}}

	if os.Args[1] != "-" {
		return do(ctx, d, strings.Join(os.Args[1:], " "))
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, lineMax), lineMax)
	for scanner.Scan() {
		if err := do(ctx, d, scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func do(ctx context.Context, d puppet.Driver, line string) error {
	out, err := d.Run(ctx, line)
	if out = strings.TrimSpace(out); out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(line), err)
	}
	return nil
}
