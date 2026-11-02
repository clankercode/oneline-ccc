package tea

import (
	"io"
	"testing"
	"testing/synctest"
	"time"
)

type lifetimeCancelReader struct {
	canceled chan struct{}
	closed   chan struct{}
}

func (r lifetimeCancelReader) Read([]byte) (int, error) { return 0, io.EOF }
func (r lifetimeCancelReader) Cancel() bool             { close(r.canceled); return true }
func (r lifetimeCancelReader) Close() error             { close(r.closed); return nil }

func TestInputReaderCloseWaitsForReadLoop(t *testing.T) {
	for _, mode := range []string{"quit", "kill", "release"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p := NewProgram(nil, WithInput(nil), WithOutput(io.Discard))
				defer p.cancel()
				r := lifetimeCancelReader{make(chan struct{}), make(chan struct{})}
				p.cancelReader = r
				p.readLoopDone = make(chan struct{})
				stopped := make(chan struct{})
				go func() {
					defer close(stopped)
					if mode == "release" {
						_ = p.ReleaseTerminal()
					} else {
						p.shutdown(mode == "kill")
					}
				}()
				<-r.canceled
				// Advance fake time beyond the old fallback while the real
				// completion barrier remains closed. This costs no wall time.
				time.Sleep(time.Second)
				select {
				case <-r.closed:
					t.Error("reader closed before input completion")
				default:
				}
				select {
				case <-stopped:
					t.Error("terminal boundary returned before input completion")
				default:
				}
				close(p.readLoopDone)
				<-stopped
				select {
				case <-r.closed:
				default:
					t.Error("joined reader was not closed")
				}
			})
		})
	}
}
