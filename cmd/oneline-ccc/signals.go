package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/clankercode/oneline-ccc/internal/core"
)

func useTUI(o flags, d dependencies) bool {
	_, noColor := d.lookup("NO_COLOR")
	term, _ := d.lookup("TERM")
	return o.command == "" && !o.help && !o.showVersion && d.terminal && !o.plain && !noColor && term != "dumb"
}
func tuiRequested(args []string, d dependencies) bool {
	o, err := parse(args)
	return err == nil && useTUI(o, d)
}
func watchSignals(parent context.Context, ui bool) (context.Context, func()) {
	if !ui {
		return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	}
	// A suspended TUI shares its foreground process group with the cooked child.
	// Consume the parent's SIGINT without cancelling its context. The child still
	// receives the terminal's SIGINT normally; raw TUI Ctrl+C is a tea key event.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	ctx, stop := signal.NotifyContext(parent, syscall.SIGTERM)
	return ctx, func() { signal.Stop(interrupts); stop() }
}

type rootBoundBackend struct {
	core.Backend
	root context.Context
}

func (b rootBoundBackend) Command(caller context.Context, action, id string) (*exec.Cmd, error) {
	if err := b.root.Err(); err != nil {
		return nil, err
	}
	// TUI command contexts outlive terminal suspension. Bind them to SIGTERM as
	// well, otherwise ExecProcess can block program shutdown waiting for a child.
	merged, cancel := context.WithCancel(caller)
	stopRoot := context.AfterFunc(b.root, cancel)
	stopCaller := context.AfterFunc(caller, func() { stopRoot(); cancel() })
	command, err := b.Backend.Command(merged, action, id)
	if err != nil || command == nil {
		stopCaller()
		stopRoot()
		cancel()
	}
	// On normal return the TUI cancels caller after the child callback. That
	// removes the root cancellation registration without a polling goroutine.
	return command, err
}
