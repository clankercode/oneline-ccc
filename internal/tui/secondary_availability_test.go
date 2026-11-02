package tui

import (
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/core"
)

type backendWithoutSSHAvailability struct{ core.Backend }

func TestBackendWithoutSSHAvailabilityKeepsHandoffUnavailable(t *testing.T) {
	f := fixture()
	f.sshAvailable = true
	m := New(backendWithoutSSHAvailability{f}, Options{NoColor: true}).(*model)
	run(t, m, m.Init())
	m.goTo("SSH")
	if m.focused().id != "back" {
		t.Fatal("backend without explicit availability offered a handoff")
	}
	m.focusID("ssh")
	if m.focused().reason == "" || press(m, "enter") != nil || m.modal != "" || f.commandAction != "" {
		t.Fatal("missing availability did not refuse without command preparation")
	}
}

func TestUnavailableSecondaryEntryUsesBackAndRestoresVisit(t *testing.T) {
	for _, screen := range []string{"SSH", "Utilities"} {
		t.Run(screen, func(t *testing.T) {
			m, f := loaded(t)
			m.goTo(screen)
			if m.focused().id != "back" {
				t.Fatalf("initial action is %q, want useful Back", m.focused().id)
			}
			if press(m, "enter") != nil {
				t.Fatal("Back returned external work")
			}
			if m.screen != "Dashboard" || f.commandAction != "" || f.serviceCalls != 0 {
				t.Fatal("initial Back did not return without external work")
			}
			m.goTo(screen)
			m.focus = navCount
			id := m.focused().id
			if m.focused().reason == "" {
				t.Fatal("unavailable action has no explanation")
			}
			m.goTo("Settings")
			m.goTo(screen)
			if m.focused().id != id {
				t.Fatal("revisit lost the explicit control location")
			}
			if press(m, "enter") != nil {
				t.Fatal("refused operation returned external work")
			}
			if m.modal != "" || f.commandAction != "" || f.serviceCalls != 0 || m.status == "" {
				t.Fatal("unavailable operation failed to explain its refusal")
			}
		})
	}
}

func TestAbsentMonitorPutsInstallationBeforeRefusedLifecycle(t *testing.T) {
	m, f := loaded(t)
	m.snapshot.Service = core.ServiceInfo{Backend: "systemd user", Status: "Not installed"}
	m.goTo("Service")
	if m.focused().id != "service:status" {
		t.Fatal("inspection must be the first action")
	}
	press(m, "down")
	if m.focused().id != "service:install" {
		t.Fatalf("next action is %q; want installation review", m.focused().id)
	}
	press(m, "enter")
	if m.modal != "service:install" || m.modalFocus != 0 || f.serviceCalls != 0 {
		t.Fatal("install review skipped safe Back or called the backend")
	}
	press(m, "esc")
	for _, id := range []string{"service:start", "service:stop", "service:restart", "service:uninstall"} {
		m.focusID(id)
		if !strings.Contains(strings.ToLower(m.focused().reason), "install") {
			t.Fatalf("%s lacks installation prerequisite", id)
		}
		if press(m, "enter") != nil {
			t.Fatal("refused lifecycle returned external work")
		}
		if f.serviceCalls != 0 || m.modal != "" {
			t.Fatalf("unavailable %s was dispatched", id)
		}
	}
}

func TestUnknownMonitorRequiresInspectionBeforeMutation(t *testing.T) {
	for _, status := range []string{"", "Unknown", "Unavailable", "Conflict", "Activating", "Deactivating"} {
		m, f := loaded(t)
		m.snapshot.Service.Status = status
		m.goTo("Service")
		for _, id := range []string{"service:start", "service:stop", "service:restart", "service:install", "service:uninstall"} {
			m.focusID(id)
			if m.focused().reason == "" {
				t.Fatalf("%q permits %s without a stable observation", status, id)
			}
			if press(m, "enter") != nil {
				t.Fatal("uncertain state returned external work")
			}
			if f.serviceCalls != 0 || m.modal != "" {
				t.Fatalf("%q dispatched %s", status, id)
			}
		}
	}
}

func TestDashboardUsesSameMonitorInstallationPrerequisite(t *testing.T) {
	for _, status := range []string{"Unknown", "Conflict", "Stopped", "Failed", "Not installed"} {
		t.Run(status, func(t *testing.T) {
			m, f := loaded(t)
			m.snapshot.Service.Status = status
			m.focusID("service:install")
			if m.focused().id != "service:install" {
				t.Fatal("Dashboard installation route is absent")
			}
			if status == "Not installed" {
				if m.focused().reason != "" {
					t.Fatal("absent monitor installation was refused")
				}
				press(m, "enter")
				if m.modal != "service:install" || m.modalFocus != 0 || f.serviceCalls != 0 {
					t.Fatal("installation did not reach safe review without work")
				}
				press(m, "tab")
				run(t, m, press(m, "enter"))
				if f.serviceCalls != 1 || f.serviceAction != "install" {
					t.Fatal("explicitly confirmed installation was not dispatched")
				}
				return
			}
			if m.focused().reason == "" {
				t.Fatalf("Dashboard offers installation for %s", status)
			}
			if press(m, "enter") != nil || m.modal != "" || f.serviceCalls != 0 {
				t.Fatal("Dashboard refused installation opened review or scheduled work")
			}
		})
	}
}
