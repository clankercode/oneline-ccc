//go:build darwin && race

package codexrpc

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// Isolated expected-red evidence reporter. Never merge into production/release.
func TestMain(m *testing.M) {
	fixture := a8ReporterFixture(os.Args)
	flag.Parse()
	verbose := "true"
	if fixture {
		verbose = "false"
	}
	if err := flag.Set("test.v", verbose); err != nil {
		fmt.Fprintln(os.Stderr, "A8 diagnostic reporter setup failed:", err)
		os.Exit(2)
	}
	if !fixture {
		fmt.Fprintln(os.Stderr, "A8_EXPECTED_DIAGNOSTIC_FAILURE: codexrpc isolated reporter; underlying test statuses follow")
	}
	code := m.Run()
	if !fixture {
		fmt.Fprintf(os.Stderr, "A8_EXPECTED_DIAGNOSTIC_FAILURE: codexrpc underlying_exit=%d; never merge/release this reporter\n", code)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func a8ReporterFixture(args []string) bool {
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "version-fixture" {
			return true
		}
		if args[i] == "--" {
			switch args[i+1] {
			case "rpc-fixture", "cleanup-rpc", "cleanup-helper":
				return true
			}
		}
	}
	return false
}
