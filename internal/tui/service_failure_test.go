package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/core"
)

type failedServiceBackend struct {
	*fakeBackend
	outcome string
}

func (f *failedServiceBackend) Service(context.Context, string) (core.ServiceInfo, error) {
	f.serviceCalls++
	return core.ServiceInfo{Backend: "systemd user", Status: f.outcome, Detail: "Preserve state; reconcile before retrying."}, errors.New("observer lifecycle did not complete; inspect native user backend state before retrying")
}
func TestFinalFixServiceFailureState(t *testing.T) {
	for _, prior := range []string{"Stopped", "Running"} {
		for _, outcome := range []string{"Unknown", "Conflict", "Unavailable"} {
			t.Run(prior+"/"+outcome, func(t *testing.T) {
				f := &failedServiceBackend{fixture(), outcome}
				f.snapshot.Service = core.ServiceInfo{Backend: "systemd user", Status: prior}
				m := New(f, Options{NoColor: true}).(*model)
				run(t, m, m.Init())
				m.goTo("Service")
				run(t, m, m.service("start"))
				if m.snapshot.Service.Status != outcome || !strings.Contains(m.View().Content, "Observed: "+outcome) {
					t.Fatalf("returned %s discarded; retained %s", outcome, m.snapshot.Service.Status)
				}
				if !strings.Contains(m.status, "failed") || !strings.Contains(m.View().Content, "reconcile") || strings.Contains(m.status, "success") || f.serviceCalls != 1 {
					t.Fatal("failure/reconciliation missing or mutation resent")
				}
			})
		}
	}
}
func TestFinalFixInvalidServiceFailureStateIsIgnored(t *testing.T) {
	f := &failedServiceBackend{fixture(), "invented"}
	f.snapshot.Service = core.ServiceInfo{Backend: "systemd user", Status: "Stopped"}
	m := New(f, Options{NoColor: true}).(*model)
	run(t, m, m.Init())
	run(t, m, m.service("start"))
	if m.snapshot.Service.Status != "Stopped" {
		t.Fatal("invalid failure state projected")
	}
}
