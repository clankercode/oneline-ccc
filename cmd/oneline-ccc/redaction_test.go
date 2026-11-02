package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestStatusJSONRedactsNestedPrivateValuesAndPreservesBackend(t *testing.T) {
	d, f, out, _, _ := harness(t)
	f.snapshot.Clients[0].Reason = "Inspect /private/codex at https://secret.invalid/v1\x1b]52;c;clipboard\a"
	f.snapshot.Receipts = []core.Receipt{{ID: "receipt", Summary: "Gateway https://secret.invalid/v1 /private/codex Bearer synthetic-token"}}
	if run(context.Background(), []string{"status", "--json"}, d) != 0 {
		t.Fatal("JSON failed")
	}
	var decoded core.Snapshot
	if json.Unmarshal(out.Bytes(), &decoded) != nil {
		t.Fatal("JSON invalid")
	}
	if strings.Contains(out.String(), "synthetic-token") || strings.Contains(out.String(), "secret.invalid") || strings.Contains(out.String(), "/private/codex") || strings.Contains(out.String(), "clipboard") {
		t.Fatal("nested private values leaked")
	}
	if f.snapshot.Clients[0].Profile.BaseURL != "https://secret.invalid/v1" || f.snapshot.Clients[0].Executable != "/private/codex" {
		t.Fatal("status mutated backend snapshot")
	}
}
func TestUnknownCredentialTextCannotBePrintedAsStatus(t *testing.T) {
	d, f, out, _, _ := harness(t)
	f.snapshot.Credential = "synthetic-raw-value"
	if run(context.Background(), []string{"status"}, d) != 0 || strings.Contains(out.String(), "synthetic-raw-value") {
		t.Fatal("unknown credential status leaked raw text")
	}
}
func TestExactModelIDNotMisclassifiedAsCredential(t *testing.T) {
	d, f, out, _, _ := harness(t)
	f.snapshot.Clients[0].Profile.Model = "sk-model-is-an-exact-id"
	if run(context.Background(), []string{"status", "--json"}, d) != 0 || !strings.Contains(out.String(), "sk-model-is-an-exact-id") {
		t.Fatal("exact model ID was rewritten")
	}
}
func TestInvalidCommandMatrixCannotOpenState(t *testing.T) {
	for _, args := range [][]string{{"service"}, {"service", "enable"}, {"launch", "pi"}, {"launch"}, {"ssh", "server"}, {"--once"}, {"status", "--once"}, {"--theme=ultraviolet"}, {"--state-dir"}, {"--ascii=invalid"}} {
		d, _, _, _, calls := harness(t)
		if run(context.Background(), args, d) != 2 || *calls != 0 {
			t.Fatalf("invalid arguments opened state: %q", args)
		}
	}
}
func TestDoctorReportsMissingCodexAndCancellationIsTruthful(t *testing.T) {
	d, f, _, _, _ := harness(t)
	f.snapshot.Clients[0].Installed = false
	if run(context.Background(), []string{"doctor"}, d) != 1 {
		t.Fatal("doctor claimed missing client ready")
	}
	f.err = context.Canceled
	if run(context.Background(), []string{"status"}, d) != 130 {
		t.Fatal("cancellation status lost")
	}
}
func TestOutputWriteFailureReturnsFailure(t *testing.T) {
	d, _, _, _, _ := harness(t)
	d.out = rejectWriter{}
	if run(context.Background(), []string{"status", "--json"}, d) == 0 {
		t.Fatal("output failure reported success")
	}
}

type rejectWriter struct{}

func (rejectWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic output failure") }
func TestRepairOutputRemainsManualAndEscapeFree(t *testing.T) {
	d, _, out, _, _ := harness(t)
	if run(context.Background(), []string{"repair"}, d) != 0 || !bytes.Contains(out.Bytes(), []byte("Manual redacted")) {
		t.Fatal("manual brief missing")
	}
}

func TestPrecancelledActionDoesNotOpenEngine(t *testing.T) {
	d, _, _, _, calls := harness(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if run(ctx, []string{"status"}, d) != 130 || *calls != 0 {
		t.Fatal("precancelled action opened application state")
	}
}
func TestHelpOutputFailureReturnsFailure(t *testing.T) {
	d, _, _, _, _ := harness(t)
	d.out = rejectWriter{}
	if run(context.Background(), []string{"--help"}, d) == 0 {
		t.Fatal("help output failure returned success")
	}
}

func TestExplicitEmptyPathCannotFallBackToOperatorDefaults(t *testing.T) {
	for _, arg := range []string{"--state-dir=", "--codex-home=", "--codex-executable="} {
		d, _, _, _, calls := harness(t)
		if run(context.Background(), []string{arg, "status"}, d) != 2 || *calls != 0 {
			t.Fatal("explicit empty path silently used a default")
		}
	}
}
func TestServiceRepairAndRecoveryOutputFailures(t *testing.T) {
	for _, args := range [][]string{{"service", "status"}, {"repair"}, {"recover"}, {"version"}} {
		d, _, _, _, _ := harness(t)
		d.out = rejectWriter{}
		if run(context.Background(), args, d) == 0 {
			t.Fatalf("write failure ignored: %v", args)
		}
	}
}

type cancelObserver struct {
	*fakeEngine
	cancel context.CancelFunc
}

func (b cancelObserver) Observe(context.Context, bool) error {
	b.cancel()
	return errors.New("backend sanitized cancellation")
}
func TestObserverCancellationKeepsInterruptionStatus(t *testing.T) {
	d, f, _, _, _ := harness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.newEngine = func(engineOptions) (engine, error) { return cancelObserver{f, cancel}, nil }
	if got := run(ctx, []string{"observe"}, d); got != 130 {
		t.Fatalf("observer cancellation became status %d", got)
	}
}
