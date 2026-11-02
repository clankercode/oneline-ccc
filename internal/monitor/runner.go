package monitor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

type cappedOutput struct {
	data     []byte
	exceeded bool
}

func (b *cappedOutput) Write(p []byte) (int, error) {
	if len(p) > recordLimit-len(b.data) {
		b.exceeded = true
		return 0, errIO
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
func runCommand(ctx context.Context, name string, args ...string) (commandResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch strings.ToUpper(key) {
		case "HOME", "USER", "LOGNAME", "PATH", "SYSTEMROOT", "WINDIR", "USERPROFILE", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "LANG", "LC_ALL", "TMPDIR", "TMP", "TEMP":
			cmd.Env = append(cmd.Env, entry)
		}
	}
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	output := &cappedOutput{}
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	result := commandResult{Output: string(output.data)}
	if ctx.Err() != nil || output.exceeded {
		return commandResult{}, errIO
	}
	if err == nil {
		return result, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		result.Code = exit.ExitCode()
		return result, nil
	}
	return commandResult{}, errUnavailable
}
