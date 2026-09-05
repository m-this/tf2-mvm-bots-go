// Command rc sends one console command to the test-bed server and prints the
// reply. It is rcon.py, without the Python. TESTBED_PORT picks the bed, as it
// does for the runner; the first bed is on 27025.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/rcon"
)

func main() {
	port := os.Getenv("TESTBED_PORT")
	if port == "" {
		port = "27025"
	}
	c := rcon.Client{Addr: "127.0.0.1:" + port, Password: "testbed"}
	out, err := c.Do(strings.Join(os.Args[1:], " "))
	fmt.Println(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
