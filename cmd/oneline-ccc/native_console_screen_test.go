package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// This ASCII VT cell reader reconstructs renderer diffs, including sequences
// split across reads. Unknown cursor operations fail closed. It is not an app
// renderer. The production journey requests --ascii and uses ASCII input only.
type nativeConsoleScreen struct {
	cells                                   [][]rune
	w, h, x, y, savedX, savedY, top, bottom int
	wrap                                    bool
	last                                    rune
	parser                                  *ansi.Parser
	err                                     error
}

func newNativeConsoleScreen(w, h int) *nativeConsoleScreen {
	s := &nativeConsoleScreen{wrap: true}
	s.resize(w, h)
	scroll := func(first, n int, down bool) {
		first = max(s.top, min(s.bottom, first))
		n = max(1, min(s.bottom-first+1, n))
		if down {
			copy(s.cells[first+n:s.bottom+1], s.cells[first:s.bottom+1-n])
			for row := first; row < first+n; row++ {
				s.cells[row] = make([]rune, s.w)
			}
		} else {
			copy(s.cells[first:s.bottom+1-n], s.cells[first+n:s.bottom+1])
			for row := s.bottom + 1 - n; row <= s.bottom; row++ {
				s.cells[row] = make([]rune, s.w)
			}
		}
	}
	newline := func() {
		if s.y == s.bottom {
			scroll(s.top, 1, false)
		} else {
			s.y = min((s.h - 1), s.y+1)
		}
	}
	print := func(r rune) {
		if s.x >= s.w {
			if s.wrap {
				s.x = 0
				newline()
			} else {
				s.x = (s.w - 1)
			}
		}
		s.cells[s.y][s.x], s.last = r, r
		s.latchKeyExposure(s.y)
		s.x++
	}
	p := ansi.NewParser()
	s.parser = p
	p.SetHandler(ansi.Handler{
		Print: print,
		Execute: func(b byte) {
			switch b {
			case '\r':
				s.x = 0
			case '\n', '\v', '\f':
				newline()
			case '\b':
				s.x = max(0, s.x-1)
			case '\t':
				s.x = min((s.w - 1), (s.x/8+1)*8)
			}
		},
		HandleEsc: func(cmd ansi.Cmd) {
			switch cmd.Final() {
			case '7':
				s.savedX, s.savedY = s.x, s.y
			case '8':
				s.x, s.y = min(s.w-1, s.savedX), min(s.h-1, s.savedY)
			case 'M':
				if s.y == s.top {
					scroll(s.top, 1, true)
				} else {
					s.y = max(0, s.y-1)
				}
			case 'D', 'E':
				newline()
				if cmd.Final() == 'E' {
					s.x = 0
				}
			}
		},
		HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
			param := func(i, def int) int { n, _, _ := params.Param(i, def); return n }
			n := max(1, param(0, 1))
			s.x = min((s.w - 1), s.x)
			switch cmd.Final() {
			case 'H', 'f':
				s.y, s.x = max(0, min((s.h-1), n-1)), max(0, min((s.w-1), param(1, 1)-1))
			case 'A':
				s.y = max(0, s.y-n)
			case 'B', 'e':
				s.y = min((s.h - 1), s.y+n)
			case 'C', 'a':
				s.x = min((s.w - 1), s.x+n)
			case 'D':
				s.x = max(0, s.x-n)
			case 'G', '`':
				s.x = min((s.w - 1), n-1)
			case 'd':
				s.y = min((s.h - 1), n-1)
			case 'I':
				s.x = min((s.w - 1), (s.x/8+n)*8)
			case 'Z':
				s.x = max(0, ((s.x-1)/8-n+1)*8)
			case 'J', 'K', 'X':
				start, end := s.y*s.w+s.x, s.h*s.w
				if cmd.Final() == 'K' {
					end = (s.y + 1) * s.w
				}
				if cmd.Final() == 'X' {
					end = min((s.y+1)*s.w, start+n)
				} else if mode := param(0, 0); mode == 1 {
					start, end = 0, start+1
					if cmd.Final() == 'K' {
						start = s.y * s.w
					}
				} else if mode >= 2 {
					start = 0
					if cmd.Final() == 'K' {
						start = s.y * s.w
					}
				}
				for i := start; i < end; i++ {
					s.cells[i/s.w][i%s.w] = 0
				}
			case 'L', 'M':
				scroll(s.y, n, cmd.Final() == 'L')
			case 'S', 'T':
				scroll(s.top, n, cmd.Final() == 'T')
			case 'P', '@':
				n = min(n, s.w-s.x)
				if cmd.Final() == 'P' {
					copy(s.cells[s.y][s.x:s.w-n], s.cells[s.y][s.x+n:])
					clear(s.cells[s.y][s.w-n:])
				} else {
					copy(s.cells[s.y][s.x+n:], s.cells[s.y][s.x:s.w-n])
					clear(s.cells[s.y][s.x : s.x+n])
				}
				s.latchKeyExposure(s.y)
			case 'b':
				for range n {
					print(s.last)
				}
			case 'r':
				s.top, s.bottom, s.x, s.y = min((s.h-1), n-1), min((s.h-1), param(1, s.h)-1), 0, 0
			case 's':
				s.savedX, s.savedY = s.x, s.y
			case 'u':
				if cmd.Prefix() == 0 { // Prefixed variants negotiate the Kitty keyboard protocol.
					s.x, s.y = min(s.w-1, s.savedX), min(s.h-1, s.savedY)
				}
			case 'h', 'l':
				if cmd.Prefix() == '?' && n == 7 {
					s.wrap = cmd.Final() == 'h'
				}
			case 'm', 'c', 'n', 'p', 'q', 't', 'W', 'g': // Styling, queries and tab-stop setup do not move cells.
			default:
				if s.err == nil {
					s.err = fmt.Errorf("unsupported terminal cursor operation %q", cmd.Final())
				}
			}
		},
	})
	return s
}

// Inspect each changed row while the parser applies updates, not after a whole
// transport read. Cursor-addressed text can expose a key and erase it in that
// same read. Checking the row avoids allocating/scanning the full screen per
// byte; print and horizontal insert/delete are the operations that can assemble
// new contiguous text. Scrolling moves whole rows, and erase only blanks cells.
func (s *nativeConsoleScreen) latchKeyExposure(y int) {
	if s.err != nil {
		return
	}
	row := s.cells[y]
	for start := 0; start+len(nativeConsoleKey) <= len(row); start++ {
		matched := true
		for i := range nativeConsoleKey { // The fixture key and terminal are ASCII.
			if row[start+i] != rune(nativeConsoleKey[i]) {
				matched = false
				break
			}
		}
		if matched {
			s.err = fmt.Errorf("dummy credential leaked into observed console cells")
			return
		}
	}
}

func (s *nativeConsoleScreen) resize(w, h int) {
	cells := make([][]rune, h)
	for y := range cells {
		cells[y] = make([]rune, w)
		if y < len(s.cells) {
			copy(cells[y], s.cells[y])
		}
	}
	s.cells, s.w, s.h, s.top, s.bottom = cells, w, h, 0, h-1
	s.x, s.y = min(s.x, w-1), min(s.y, h-1)
}
func (s *nativeConsoleScreen) feed(data []byte) {
	for _, b := range data {
		s.parser.Advance(b)
	}
}
func (s *nativeConsoleScreen) text() string {
	var out strings.Builder
	for _, row := range s.cells {
		for _, r := range row {
			if r == 0 {
				r = ' '
			}
			out.WriteRune(r)
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func TestNativeConsoleDriverControls(t *testing.T) {
	t.Run("cell-diffs-split-sequences-and-focus", func(t *testing.T) {
		s := newNativeConsoleScreen(120, 40)
		for _, chunk := range []string{"\x1b[HSession-only storage\x1b[", "3Garch:\x1b[K\r\n  First\r\n> Second\x1b[2;1H>", "\x1b[3;1H "} {
			s.feed([]byte(chunk))
		}
		if s.err != nil || !nativeConsoleFrameHas(s.text(), "Search:", "> First") || strings.Contains(s.text(), "> Second") || strings.Contains(s.text(), "Session-only") {
			t.Fatalf("incorrect reconstructed cells: %v %q", s.err, s.text())
		}
		if nativeConsoleFrameHas(s.text(), "> Second") {
			t.Fatal("stale focus accepted")
		}
		s.feed([]byte("\x1b[2J\x1b[H> [x] Manage client"))
		if !nativeConsoleFrameHas(s.text(), "> [x] Manage client") || nativeConsoleFrameHas(s.text(), "> [ ] Manage client") {
			t.Fatal("checkbox mutation was not detected")
		}
		s.resize(40, 12)
		s.feed([]byte("\x1b[2J\x1b[12;1Hresize sentinel"))
		if len(strings.Split(strings.TrimSuffix(s.text(), "\n"), "\n")) != 12 || !strings.Contains(s.text(), "resize sentinel") {
			t.Fatal("resize/cursor controls lost")
		}
		s.feed([]byte("\x1b[1z"))
		if s.err == nil {
			t.Fatal("unknown cursor operation accepted")
		}
	})
	t.Run("mask-color-and-entry-negative-controls", func(t *testing.T) {
		for _, raw := range []string{nativeConsoleKey, "\x1b[H" + nativeConsoleKey, "\x1b[31m", "\x1b[44m", "\x1b[38;2;1;2;3m", "\x1b[48:2::1:2:3m", "\x1b[58;5;4m", "\x1b[91m", "\x1b[107m"} {
			if nativeConsoleOutputError([]byte(raw)) == nil {
				t.Fatalf("regression injection accepted: %q", raw)
			}
		}
		if err := nativeConsoleOutputError([]byte("\x1b[?1049h\x1b[2J\x1b[0;39;49;59mAPI key: [hidden]")); err != nil {
			t.Fatal(err)
		}
		s := newNativeConsoleScreen(120, 40)
		s.feed([]byte("\x1b[H" + nativeConsoleKey[:10] + "\x1b[2;1Hfooter\x1b[1;11H" + nativeConsoleKey[10:]))
		if nativeConsoleOutputError([]byte(s.text())) == nil {
			t.Fatal("credential assembled by cursor edits was not detected")
		}
		if nativeConsoleFrameHas("Local inspection | Gateway A", "Inspected local settings; drafts retained") {
			t.Fatal("plain fallback accepted as interactive entry")
		}
	})
	t.Run("input-stack-control", func(t *testing.T) {
		if nativeInputStacks("goroutine 7 [running]:\nuv.(*TerminalReader).sendEvents(0x1)\ncreated by ignored") == "" {
			t.Fatal("live decoder was not detected")
		}
		if nativeInputStacks("goroutine 7 [running]:\nfinished()\ncreated by uv.(*TerminalReader).StreamEvents(0x1)") != "" {
			t.Fatal("creation metadata mistaken for active input")
		}
	})
}

func TestNativeConsoleTransientKeyExposure(t *testing.T) {
	prefix := "\x1b[H" + nativeConsoleKey[:10] + "\x1b[2;1Hfooter\x1b[1;11H" + nativeConsoleKey[10:]
	// Establish that the injected cursor updates really display the whole key.
	exposed := newNativeConsoleScreen(120, 40)
	exposed.feed([]byte(prefix))
	if !strings.Contains(exposed.text(), nativeConsoleKey) {
		t.Fatal("invalid injection: cursor updates did not assemble the key")
	}
	for _, erase := range []struct{ name, bytes string }{
		{"erase-line", "\x1b[1;1H\x1b[2K"},
		{"clear-screen", "\x1b[2J\x1b[HAPI key: [hidden]"},
		{"alternate-screen", "\x1b[?1049l\x1b[2J\x1b[?1049h\x1b[HAPI key: [hidden]"},
	} {
		t.Run(erase.name, func(t *testing.T) {
			raw := []byte(prefix + erase.bytes)
			if err := nativeConsoleOutputError(raw); err != nil {
				t.Fatalf("invalid injection: raw/stripped oracle already detects it: %v", err)
			}
			for _, split := range []bool{false, true} {
				s := newNativeConsoleScreen(120, 40)
				if split {
					s.feed([]byte(prefix[:len(prefix)-5]))
					s.feed([]byte(prefix[len(prefix)-5:] + erase.bytes))
				} else {
					s.feed(raw)
				}
				if strings.Contains(s.text(), nativeConsoleKey) {
					t.Fatal("invalid injection: final cells still contain key")
				}
				if s.err == nil {
					t.Errorf("split=%t: transient cursor-assembled exposure was not latched", split)
					continue
				}
				latched := s.err
				s.resize(40, 12)
				s.feed([]byte("\x1b[2J\x1b[?1049l\x1b[?1049h\x1b[HAPI key: [hidden]"))
				s.resize(120, 40)
				s.feed([]byte("\x1b[1z")) // Even a later parser failure must not replace it.
				if s.err != latched {
					t.Fatal("erase, resize or alternate-screen transition cleared the exposure latch")
				}
			}
		})
	}
	t.Run("clean-erase-and-resize", func(t *testing.T) {
		s := newNativeConsoleScreen(120, 40)
		s.feed([]byte("\x1b[HAPI key: [hidden]\x1b[2J"))
		s.resize(40, 12)
		s.feed([]byte("\x1b[?1049l\x1b[?1049h\x1b[Hclean"))
		s.resize(120, 40)
		if s.err != nil {
			t.Fatal("clean updates rejected:", s.err)
		}
	})
	t.Run("delete-assembles-then-clears", func(t *testing.T) {
		raw := []byte("\x1b[H" + nativeConsoleKey[:10] + "_" + nativeConsoleKey[10:] + "\x1b[1;11H\x1b[P\x1b[2J")
		if err := nativeConsoleOutputError(raw); err != nil {
			t.Fatal("invalid delete injection:", err)
		}
		s := newNativeConsoleScreen(120, 40)
		s.feed(raw)
		if s.err == nil || strings.Contains(s.text(), nativeConsoleKey) {
			t.Fatal("horizontal delete exposure was not latched before clear")
		}
	})
	t.Run("ordinary-read-and-final-drain-share-oracle", func(t *testing.T) {
		raw := []byte(prefix + "\x1b[2J\x1b[HAPI key: [hidden]")
		ordinary := &nativeConsoleDriver{screen: newNativeConsoleScreen(120, 40)}
		if err := ordinary.consume(raw); err == nil {
			t.Fatal("ordinary read oracle accepted transient exposure")
		}
		final := &nativeConsoleDriver{screen: newNativeConsoleScreen(120, 40), host: nativeConsoleDrainControl{tail: raw}}
		if err := final.consumeDrain(); err == nil {
			t.Fatal("final drain oracle accepted transient exposure")
		}
		if err := final.consume([]byte("\x1b[2Jclean")); err == nil {
			t.Fatal("later clean batch cleared the exposure latch")
		}
		split := &nativeConsoleDriver{screen: newNativeConsoleScreen(120, 40), host: nativeConsoleDrainControl{tail: []byte(prefix[len(prefix)-5:] + "\x1b[2J")}}
		if err := split.consume([]byte(prefix[:len(prefix)-5])); err != nil {
			t.Fatal("incomplete ordinary fragment rejected:", err)
		}
		if err := split.consumeDrain(); err == nil {
			t.Fatal("key assembled across ordinary read and final drain was not latched")
		}
		clean := &nativeConsoleDriver{screen: newNativeConsoleScreen(120, 40), host: nativeConsoleDrainControl{tail: []byte("\x1b[2JAPI key: [hidden]")}}
		if err := clean.consumeDrain(); err != nil {
			t.Fatal("clean final drain rejected:", err)
		}
	})
}

// Only final-drain transport data is substituted in this shared assertion
// control. Calling any other host method would fail through the nil interface.
type nativeConsoleDrainControl struct {
	nativeConsoleHost
	tail []byte
}

func (h nativeConsoleDrainControl) drained() []byte { return h.tail }
