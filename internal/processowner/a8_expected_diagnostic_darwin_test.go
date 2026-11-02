//go:build darwin && race

package processowner

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	for _, arg := range os.Args {
		if arg == "owned-cleanup-fixture" {
			// Helpers speak a small stdout protocol; keep their test runner quiet.
			os.Exit(m.Run())
		}
	}
	// Successful nested subtests otherwise discard their buffered t.Log output.
	if err := flag.Set("test.v", "true"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// This isolated diagnostic branch intentionally fails only the Darwin race
// gate, so the existing non-verbose CI job retains native characterization logs.
// Never merge this probe into main or a release branch.
func TestA8ExpectedNativeDiagnostic(t *testing.T) {
	t.Run("zombie-only", TestDarwinZombieOnlyGroupDiagnostic)
	t.Run("root-wait", TestPOSIXCleanupAcrossRootWait)
	t.Fatal("A8_EXPECTED_DIAGNOSTIC_FAILURE: isolated evidence probe; not a release gate")
}
