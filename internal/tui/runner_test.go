package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type runnerBackend struct {
	*fakeBackend
	command *exec.Cmd
}

func (f *runnerBackend) Command(context.Context, string, string) (*exec.Cmd, error) {
	return f.command, nil
}
func TestFinalFixRunnerChildFixture(t *testing.T) {
	if os.Getenv("ONELINE_TUI_RUNNER_FIXTURE") == "1" {
		os.Exit(23)
	}
}
func TestFinalFixTerminalRunner(t *testing.T) {
	child := exec.Command(os.Args[0], "-test.run=^TestFinalFixRunnerChildFixture$", "--", "literal argument with spaces", "模型;%$x")
	child.Env = []string{"ONELINE_TUI_RUNNER_FIXTURE=1"}
	called := false
	runner := func(ctx context.Context, cmd *exec.Cmd, in io.Reader, out, errout io.Writer) error {
		called = true
		if ctx.Err() != nil || cmd != child || !reflect.DeepEqual(cmd.Env, []string{"ONELINE_TUI_RUNNER_FIXTURE=1"}) || !reflect.DeepEqual(cmd.Args[2:], []string{"--", "literal argument with spaces", "模型;%$x"}) || in == nil || out == nil || errout == nil {
			t.Error("runner lost context/command/streams/literal arguments or explicit environment")
		}
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = errout
		return cmd.Run()
	}
	m := New(&runnerBackend{fixture(), child}, Options{NoColor: true, Runner: runner}).(*model)
	run(t, m, m.Init())
	m.initial = m.command("launch")
	var returned error
	filter := func(_ tea.Model, msg tea.Msg) tea.Msg {
		if r, ok := msg.(result); ok && r.kind == "child" {
			returned = r.err
			return tea.QuitMsg{}
		}
		return msg
	}
	_, err := tea.NewProgram(m, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithFilter(filter)).Run()
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("terminal handoff bypassed injected owned-child runner")
	}
	var exit *exec.ExitError
	if !errors.As(returned, &exit) || exit.ExitCode() != 23 {
		t.Fatal("child exit status lost")
	}
}
