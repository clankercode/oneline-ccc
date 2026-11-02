package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestKeyMaskConstantAndSessionOnlyDefault(t *testing.T) {
	m, f := loaded(t)
	m.goTo("Gateway")
	activate(t, m, "key")
	empty := m.View().Content
	m.Update(tea.PasteMsg{Content: "synthetic-key-with-distinct-length"})
	if got := m.View().Content; got != empty {
		t.Fatalf("key rendering changes with length: %q", got)
	}
	press(m, "enter")
	run(t, m, press(m, "enter"))
	if f.keyCalls != 1 || f.persistent || m.editor != nil {
		t.Fatal("key was not session-only or buffer retained")
	}
	activate(t, m, "key")
	m.Update(tea.PasteMsg{Content: "throwaway"})
	press(m, "esc")
	if f.keyCalls != 1 || strings.Contains(m.View().Content, "throwaway") {
		t.Fatal("cancel leaked or stored key")
	}
}
func TestTrustedTierEvidenceAndCodexOnly(t *testing.T) {
	m, f := loaded(t)
	if m.draft().Profile.TiersConfirmed || m.dirty() {
		t.Fatal("unmanaged evidence trusted or creates a dirty draft")
	}
	activate(t, m, "client:pi")
	activate(t, m, "enabled")
	activate(t, m, "plan")
	activate(t, m, "advanced:model")
	if f.planCalls != 0 || m.drafts["pi"].Enabled || m.editor != nil {
		t.Fatal("non-Codex client mutated")
	}
	f.snapshot.Clients[0].Managed = true
	f.snapshot.Clients[0].Profile.TiersConfirmed = true
	fresh := New(f, Options{NoColor: true}).(*model)
	run(t, fresh, fresh.Init())
	fresh.goTo("Models")
	if !strings.Contains(fresh.View().Content, "Available tiers: Up to Ultrafast") || fresh.draft().Profile.Tier != "Standard" {
		t.Fatal("trusted availability/default tier not separated")
	}
}
func TestServiceAndSSHExplicitConfirmation(t *testing.T) {
	m, f := loaded(t)
	m.snapshot.Service.Status = "Not installed"
	f.sshAvailable = true
	m.goTo("Service")
	activate(t, m, "service:install")
	if f.serviceCalls != 0 || m.modalFocus != 0 {
		t.Fatal("installation runs before confirmation")
	}
	press(m, "enter")
	if f.serviceCalls != 0 || m.modal != "" {
		t.Fatal("Back applied service action")
	}
	activate(t, m, "service:install")
	press(m, "tab")
	run(t, m, press(m, "enter"))
	if f.serviceCalls != 1 || f.serviceAction != "install" {
		t.Fatal("confirmed service not wired")
	}
	m.goTo("SSH")
	activate(t, m, "ssh")
	if m.modal != "ssh" || m.modalFocus != 0 {
		t.Fatal("available SSH did not open safe review")
	}
	press(m, "esc")
	if f.commandAction != "" {
		t.Fatal("SSH invoked without acceptance")
	}
	activate(t, m, "ssh")
	press(m, "tab")
	cmd := press(m, "enter")
	if cmd == nil {
		t.Fatal("confirmed available SSH did not prepare a command")
	}
	_ = cmd() // Prepare only; never execute the returned command.
	if f.commandAction != "ssh" {
		t.Fatal("confirmed SSH did not reach the backend")
	}
}
func TestPublicViewsNeverExposeRoutesExecutablesOrEscapes(t *testing.T) {
	m, f := loaded(t)
	f.snapshot.Clients[0].Status = "\x1b]52;c;clipboard\aUnverified\x1b[31m"
	run(t, m, press(m, "r"))
	for _, screen := range []string{"Dashboard", "Models", "Review", "Jobs", "Gateway", "Service", "Repair", "SSH", "Utilities", "Settings"} {
		m.goTo(screen)
		v := m.View().Content
		if strings.Contains(v, "private.invalid") || strings.Contains(v, "/private/executable") || strings.Contains(v, "\x1b") || strings.Contains(v, "clipboard") {
			t.Fatalf("public view leaks on %s: %q", screen, v)
		}
	}
	m.goTo("Gateway")
	activate(t, m, "advanced:endpoint")
	if !strings.Contains(m.View().Content, "https://private.invalid/v1") {
		t.Fatal("local endpoint editor omitted actual route")
	}
}
func TestStandardDoesNotAssertAcceleratedCapabilities(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	run(t, m, activate(t, m, "plan"))
	if f.planned.Profile.TiersConfirmed {
		t.Fatal("Standard plan claims accelerated capability")
	}
}

type blockedBackend struct {
	*fakeBackend
	started chan context.Context
	release chan struct{}
}

func (b *blockedBackend) Plan(ctx context.Context, id string, d core.Desired) (core.Plan, error) {
	b.started <- ctx
	select {
	case <-ctx.Done():
		return core.Plan{}, ctx.Err()
	case <-b.release:
		return b.fakeBackend.Plan(ctx, id, d)
	}
}
func TestBusyCancelWaitsForBackendAndPreservesDraft(t *testing.T) {
	f := fixture()
	b := &blockedBackend{f, make(chan context.Context, 1), make(chan struct{})}
	m := New(b, Options{NoColor: true}).(*model)
	run(t, m, m.Init())
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	activate(t, m, "tiers")
	draft := m.draft()
	cmd := activate(t, m, "plan")
	if cmd == nil {
		t.Fatal("no plan request")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	var ctx context.Context
	select {
	case ctx = <-b.started:
	case <-time.After(time.Second):
		t.Fatal("backend not entered")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 21*time.Second {
		t.Fatal("action not bounded")
	}
	if m.busy != "plan" {
		t.Fatal("missing busy state")
	}
	activate(t, m, "model:glm-5.3")
	if m.draft() != draft {
		t.Fatal("busy action allowed draft mutation")
	}
	press(m, "ctrl+c")
	if m.modal != "quit" || ctx.Err() != nil {
		t.Fatal("quit decision missing")
	}
	press(m, "tab")
	press(m, "enter")
	if ctx.Err() == nil || m.busy == "" {
		t.Fatal("cancel did not wait for backend final outcome")
	}
	select {
	case msg := <-done:
		m.Update(msg)
	case <-time.After(time.Second):
		t.Fatal("cancellation did not return")
	}
	if m.busy != "" || m.draft() != draft || f.applyCalls != 0 {
		t.Fatal("cancel lost draft or applied")
	}
}

func TestAcceleratedTierRequiresExplicitRouteConfirmation(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	activate(t, m, "tier:Ultrafast")
	activate(t, m, "capacity")
	activate(t, m, "plan")
	if f.planCalls != 0 {
		t.Fatal("unverified accelerated tier accepted")
	}
	activate(t, m, "tiers")
	run(t, m, activate(t, m, "plan"))
	if !f.planned.Profile.TiersConfirmed || f.planned.Profile.Model != "gpt-6.1-sol" {
		t.Fatal("confirmation or exact model lost")
	}
	press(m, "esc")
	activate(t, m, "budget:272000")
	if m.draft().Profile.TiersConfirmed || m.draft().Profile.CapacityConfirmed {
		t.Fatal("budget change did not reset attestations")
	}
}
func TestPaletteDisabledReasonAndFocusRestoration(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:pi")
	focus := m.focus
	press(m, "ctrl+p")
	m.Update(tea.PasteMsg{Content: "Review selected"})
	press(m, "enter")
	if m.modal != "palette" || !strings.Contains(m.status, "Codex") {
		t.Fatal("disabled palette action lacked reason or dismissed")
	}
	press(m, "esc")
	if m.focus != focus {
		t.Fatal("palette did not restore exact focus")
	}
}
