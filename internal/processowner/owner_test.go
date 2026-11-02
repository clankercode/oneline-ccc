package processowner

import "os/exec"

// Preparation-only policy tests never execute this synthetic path.
func childFixture(mode string) *exec.Cmd {
	cmd := exec.Command("synthetic-native-client", mode)
	cmd.Env = []string{}
	return cmd
}
