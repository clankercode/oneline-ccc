package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestLaunchRefusesConfigDriftAcrossInspection(t *testing.T) {
	e, a, _, o := fixture(t, false)
	path := filepath.Join(o.CodexHome, "config.toml")
	if err := configtxn.PrivateFile(path, []byte("original")); err != nil {
		t.Fatal(err)
	}
	a.inspect = func(context.Context) (core.Client, error) {
		if err := os.WriteFile(path, []byte("changed while inspecting"), 0600); err != nil {
			t.Fatal(err)
		}
		return core.Client{Installed: true, Supported: true, Version: "0.166.0", Profile: profile()}, nil
	}
	if command, err := e.Command(context.Background(), "launch", "codex"); command != nil || err == nil {
		t.Fatal("launch accepted a configuration change during inspection")
	}
}

type fixtureMonitor struct {
	actions  []string
	observed bool
	status   string
}

func (m *fixtureMonitor) Status(context.Context) core.ServiceInfo {
	status := m.status
	if status == "" {
		status = "Running"
	}
	return core.ServiceInfo{Backend: "systemd user", Status: status, Detail: "fixture-private-key"}
}
func (m *fixtureMonitor) Action(_ context.Context, action string) (core.ServiceInfo, error) {
	m.actions = append(m.actions, action)
	return m.Status(context.Background()), nil
}
func (m *fixtureMonitor) Observe(context.Context, bool) error { m.observed = true; return nil }
func TestMonitorDelegatesOnlyExplicitActionsAndRedactsDetail(t *testing.T) {
	e, _, _, _ := fixture(t, false)
	m := &fixtureMonitor{}
	e.deps.monitor = m
	s, err := e.Inspect(context.Background())
	if err != nil || len(m.actions) != 0 || m.observed {
		t.Fatal("inspection activated observer")
	}
	if strings.Contains(s.Service.Detail, "fixture-private-key") {
		t.Fatal("observer detail leaked private external text")
	}
	service, err := e.Service(context.Background(), "start")
	if err != nil || service.Status != "Running" || len(m.actions) != 1 {
		t.Fatal("service action not delegated")
	}
	if err = e.Observe(context.Background(), true); err != nil || !m.observed {
		t.Fatal("observation not delegated")
	}
}
func TestFailedMonitorStatusSurvivesSafeProjection(t *testing.T) {
	e, _, _, _ := fixture(t, false)
	e.deps.monitor = &fixtureMonitor{status: "Failed"}
	snapshot, err := e.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	status, err := e.Service(context.Background(), "status")
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range []core.ServiceInfo{snapshot.Service, status} {
		if service.Status != "Failed" || service.Backend != "systemd user" {
			t.Fatalf("monitor failure lost in projection: %#v", service)
		}
		if !strings.Contains(service.Detail, "failed") || strings.Contains(service.Detail, "fixture-private-key") {
			t.Fatal("failed status needs locally generated safe detail")
		}
	}
}
func TestPlanningRechecksStateAfterStagingWait(t *testing.T) {
	e, a, _, o := fixture(t, false)
	a.prepare = func(context.Context, core.Profile, []byte) ([]configtxn.Write, error) {
		if err := configtxn.PrivateFile(filepath.Join(o.StateRoot, "settings.json"), []byte("invalid fixture state")); err != nil {
			t.Fatal(err)
		}
		return []configtxn.Write{{Path: filepath.Join(o.CodexHome, "config.toml"), Mode: 0600, Data: []byte("fixture = true\n")}}, nil
	}
	if p, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: profile()}); err == nil || p.ID != "" {
		t.Fatal("plan published after protected state became invalid")
	}
}
