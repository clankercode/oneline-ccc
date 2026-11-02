package uv

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

type eofSignalReader struct {
	*strings.Reader
	eof chan struct{}
}

func (r eofSignalReader) Read(buf []byte) (int, error) {
	n, err := r.Reader.Read(buf)
	if err == io.EOF {
		close(r.eof)
	}
	return n, err
}

func TestInputDispatchCancellation(t *testing.T) {
	for _, payload := range []string{strings.Repeat("a", 1024), "\x1b"} {
		t.Run(payload[:1], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := eofSignalReader{strings.NewReader(payload), make(chan struct{})}
			d := NewTerminalReader(r, "dumb")
			done := make(chan error, 1)
			// Deliberately leave event delivery blocked, both during ordinary
			// decoding and when flushing an incomplete sequence after EOF.
			go func() { done <- d.StreamEvents(ctx, make(chan Event)) }()
			<-r.eof
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("decoder did not join after cancellation with blocked output")
			}
		})
	}
}
