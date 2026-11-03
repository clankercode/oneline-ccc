package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

const noColorMaskedFrame = "\x1b[2J\x1b[HEditing key\r\n> API key: [hidden]"

func noColorTransientParts() (string, string) {
	// The row-three text separates the two key fragments in both raw and
	// stripped bytes. Cursor positioning assembles the key on row two instead.
	first := "\x1b[HEditing key\x1b[2;1H" + noColorDummyKey[:10] + "\x1b[3;1Htransport-gap\x1b[2;11H"
	return first, noColorDummyKey[10:] + noColorMaskedFrame
}

func noColorInsertModePayload() string {
	return "\x1b[H" + noColorDummyKey[10:] +
		"\x1b[3;1Htransport-gap\x1b[4h\x1b[1;1H" +
		noColorDummyKey[:10] + "\x1b[4l" + noColorMaskedFrame
}

func TestNoColorTerminalUnsupportedModes(t *testing.T) {
	t.Run("insert-mode-fails-closed", func(t *testing.T) {
		s := newNoColorScreen()
		s.feed([]byte("\x1b[4h"))
		if s.err == nil {
			t.Fatal("unimplemented standard insert mode was silently accepted")
		}
	})
	t.Run("insert-mode-transient-assembly", func(t *testing.T) {
		// Independent insertion reference: each prefix character shifts the
		// previously displayed suffix right. This does not use the VT oracle.
		var reference [120]rune
		copy(reference[:], []rune(noColorDummyKey[10:]))
		for column, r := range []rune(noColorDummyKey[:10]) {
			copy(reference[column+1:], reference[column:119])
			reference[column] = r
		}
		if string(reference[:len(noColorDummyKey)]) != noColorDummyKey {
			t.Fatal("invalid insertion reference")
		}
		raw := noColorInsertModePayload()
		if strings.Contains(raw, noColorDummyKey) || strings.Contains(ansi.Strip(raw), noColorDummyKey) {
			t.Fatal("invalid insertion probe: raw/stripped checks already detect it")
		}
		s := newNoColorScreen()
		s.feed([]byte(raw))
		if strings.Contains(s.text(), noColorDummyKey) || !strings.Contains(s.text(), "> API key: [hidden]") {
			t.Fatal("invalid insertion probe: final cells were not cleared")
		}
		if s.err == nil {
			t.Fatal("transient insertion exposure escaped rejection")
		}
	})
}

// External controls run the real read/finish methods. A separate plain fixture
// supplies the PTY and genuine backend/terminal completion checks; only the
// additional oracle-control output is injected through its retained slave fd.
func TestNoColorTerminalTransientFixture(t *testing.T) {
	mode := os.Getenv("ONELINE_NO_COLOR_ORACLE_MODE")
	if mode == "" {
		return
	}
	p := startNoColorPTY(t, "plain", "1")
	for !strings.Contains(string(p.transcript), "PASS") {
		if remaining := time.Until(p.deadline); remaining > 0 {
			p.read(t, remaining)
		} else {
			t.Fatal("plain fixture did not complete before oracle injection")
		}
	}
	first, last := noColorTransientParts()
	if mode == "masked" {
		first, last = "", noColorMaskedFrame
	}
	if mode == "unsupported" {
		first, last = "", "\x1b[1z"+noColorMaskedFrame
	}
	if mode == "delete-then-clear" {
		first = ""
		last = "\x1b[H" + noColorDummyKey[:10] + "_" + noColorDummyKey[10:] + "\x1b[1;11H\x1b[P" + noColorMaskedFrame
	}
	if mode == "insert-mode-read" || mode == "insert-mode-final-drain" {
		first, last = "", noColorInsertModePayload()
	}
	if raw := first + last; strings.Contains(raw, noColorDummyKey) || strings.Contains(ansi.Strip(raw), noColorDummyKey) {
		t.Fatal("invalid control: raw/stripped checks already find the key")
	}
	write := func(data string) {
		t.Helper()
		if _, err := p.slave.WriteString(data); err != nil {
			t.Fatal(err)
		}
	}
	switch mode {
	case "split-read", "ordinary-final-drain":
		write(first)
		p.read(t, time.Until(p.deadline))
		write(last)
		if mode == "split-read" {
			p.read(t, time.Until(p.deadline))
		}
	case "one-read", "delete-then-clear", "masked", "unsupported", "insert-mode-read":
		write(first + last)
		p.read(t, time.Until(p.deadline))
	case "final-drain", "insert-mode-final-drain":
		write(first + last)
	default:
		t.Fatal("unknown oracle control")
	}
	p.finish(t, "ENTRY stdinTTY=true stdoutTTY=true signalUI=false")
	p.assertKeyHidden(t, "> API key: [hidden]")
}

func TestNoColorTerminalTransientReadControls(t *testing.T) {
	for _, mode := range []string{"one-read", "split-read", "final-drain", "ordinary-final-drain", "delete-then-clear", "masked", "unsupported", "insert-mode-read", "insert-mode-final-drain"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNoColorTerminalTransientFixture$")
			c.Env = []string{"ONELINE_NO_COLOR_ORACLE_MODE=" + mode, "GOMAXPROCS=1", "TMPDIR=" + t.TempDir()}
			output, err := c.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal("oracle fixture deadline exceeded")
			}
			if mode == "masked" {
				if err != nil {
					t.Fatalf("valid masked control rejected: %v\n%s", err, output)
				}
				return
			}
			want := "dummy credential exposed in terminal cells"
			if mode == "unsupported" || strings.HasPrefix(mode, "insert-mode-") {
				want = "unsupported terminal cursor operation"
			}
			if err == nil {
				t.Fatal("oracle accepted output that must be rejected: " + mode)
			}
			if !strings.Contains(string(output), want) {
				t.Fatalf("wrong oracle rejection: %v\n%s", err, output)
			}
		})
	}
}

func TestNoColorTerminalModeCommands(t *testing.T) {
	for _, sequence := range []string{
		"\x1b[4h", "\x1b[4l", "\x1b[?4h", "\x1b[?25;4h", "\x1b[?7;4l",
		"\x1b[?25:4h", "\x1b[?25$h", "\x1b[?h", "\x1b[?;25h", "\x1b[>25h",
		"\x1b[>1;4u", "\x1b[?4u", "\x1b[4u", "\x1b[>4;2;4m", "\x1b[!p",
		"\x1b[?4$p", "\x1b[8;12;40t", "\x1b[?5;4W", "\x1b[3g", "\x1b[?4H",
	} {
		t.Run("reject-"+sequence, func(t *testing.T) {
			s := newNoColorScreen()
			s.feed([]byte(sequence))
			if s.err == nil || !strings.Contains(s.err.Error(), "unsupported terminal cursor operation") {
				t.Fatal("unsupported full command was accepted")
			}
			first := s.err
			s.feed([]byte("\x1b[H" + noColorDummyKey + noColorMaskedFrame))
			if s.err != first {
				t.Fatal("later exposure/clearing replaced the first unsupported-mode error")
			}
		})
	}
	t.Run("observed-negotiation-and-visibility", func(t *testing.T) {
		s := newNoColorScreen()
		s.feed([]byte("\x1b[?2026$p\x1b[?2027$p\x1b[>4m\x1b[?1049h\x1b[?25l\x1b[?5W\x1b[?2004h\x1b[>4;2m\x1b[>1u\x1b[?u\x1b[<1u\x1b[?1049l\x1b[?25h\x1b[?2004l" + noColorMaskedFrame))
		if s.err != nil || !strings.Contains(s.text(), "> API key: [hidden]") {
			t.Fatal("observed harmless commands rejected:", s.err)
		}
	})
	t.Run("modeled-wrap-in-multi-parameter-command", func(t *testing.T) {
		s := newNoColorScreen()
		s.feed([]byte("\x1b[?25;7l\x1b[1;120HAB"))
		if s.err != nil || s.cells[0][119] != 'B' || s.cells[1][0] != 0 {
			t.Fatal("validated mode list did not apply its modeled wrap parameter")
		}
	})
}

func TestNoColorTerminalExposureLatch(t *testing.T) {
	first, last := noColorTransientParts()
	prefix := first + noColorDummyKey[10:]
	visible := newNoColorScreen()
	visible.feed([]byte(prefix))
	if !strings.Contains(visible.text(), noColorDummyKey) || visible.err == nil {
		t.Fatal("control did not visibly assemble and latch the key")
	}
	for _, tc := range []struct {
		name  string
		split int
	}{
		{"one-batch", 0},
		{"split-cursor-sequence", len(first) - 2},
		{"split-key", len(first) + 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newNoColorScreen()
			raw := first + last
			s.feed([]byte(raw[:tc.split]))
			if s.err != nil {
				t.Fatal("incomplete control rejected before the key was assembled")
			}
			s.feed([]byte(raw[tc.split:]))
			if s.err == nil || strings.Contains(s.text(), noColorDummyKey) || !strings.Contains(s.text(), "> API key: [hidden]") {
				t.Fatal("cleared exposure was not retained alongside the final masked frame")
			}
			latched := s.err
			for _, clear := range []string{"\x1b[2;1H\x1b[2K", "\x1b[2J", "\x1b[?1049l\x1b[?1049h"} {
				s.feed([]byte(clear + noColorMaskedFrame))
				if s.err != latched {
					t.Fatal("later clearing or alternate-screen controls replaced the exposure latch")
				}
			}
			s.feed([]byte("\x1b[1z"))
			if s.err != latched {
				t.Fatal("later unsupported cursor operation replaced the exposure latch")
			}
		})
	}
}
