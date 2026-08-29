// Command rc sends one console command to the test-bed server and prints the
// reply. It is rcon.py, without the Python.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/rcon"
)

func main() {
	c := rcon.Client{Addr: "127.0.0.1:27025", Password: "testbed"}
	out, err := c.Do(strings.Join(os.Args[1:], " "))
	fmt.Println(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
