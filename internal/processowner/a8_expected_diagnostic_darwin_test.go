//go:build darwin && race

package processowner

import "testing"

// This isolated diagnostic branch intentionally fails only the Darwin race
// gate, so the existing non-verbose CI job retains native characterization logs.
// Never merge this probe into main or a release branch.
func TestA8ExpectedNativeDiagnostic(t *testing.T) {
	t.Run("zombie-only", TestDarwinZombieOnlyGroupDiagnostic)
	t.Run("root-wait", TestPOSIXCleanupAcrossRootWait)
	t.Fatal("A8_EXPECTED_DIAGNOSTIC_FAILURE: isolated evidence probe; not a release gate")
}
