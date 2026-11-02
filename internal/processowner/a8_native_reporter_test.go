//go:build darwin && race

package processowner

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
		fmt.Fprintln(os.Stderr, "A8_EXPECTED_DIAGNOSTIC_FAILURE: processowner isolated reporter; underlying test statuses follow")
	}
	code := m.Run()
	if !fixture {
		fmt.Fprintf(os.Stderr, "A8_EXPECTED_DIAGNOSTIC_FAILURE: processowner underlying_exit=%d; never merge/release this reporter\n", code)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func a8ReporterFixture(args []string) bool {
	return len(args) >= 3 && args[len(args)-2] == "owned-cleanup-fixture"
}
