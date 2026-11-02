package tui

import (
	"errors"
	"fmt"
	"strings"
)

var errUnavailable = errors.New("backend did not provide a command")

type control struct{ id, label, reason string }

func (m *model) controls() []control {
	cs := []control{{"nav:Dashboard", "Dashboard", ""}, {"nav:Models", "Models", ""}, {"nav:Review", "Review", ""}, {"nav:Jobs", "Jobs", ""}}
	add := func(id, label, reason string) { cs = append(cs, control{id, label, reason}) }
	why := m.editable()
	d := m.draft()
	switch m.screen {
	case "Dashboard":
		for _, c := range m.snapshot.Clients {
			if m.matches(c.ID, c.Name) {
				check := "[ ]"
				if m.drafts[c.ID].Enabled {
					check = "[x]"
				}
				label := fmt.Sprintf("%s %s  %s", check, c.Name, c.Status)
				if !c.Installed {
					label += " | Not installed"
				}
				if c.ID == m.selected {
					label += " | Selected"
				}
				if m.drafts[c.ID] != m.baselines[c.ID] {
					label += " | Draft"
				}
				add("client:"+c.ID, label, "")
			}
		}
		if len(cs) == 4 {
			add("filter:clear", "Clear filter", "")
		}
		add("configure", "Configure selected client", noClient(m))
		add("plan", "Review selected draft", why)
		add("launch", "Launch selected client", why)
		if !m.observerDismissed && m.snapshot.Service.Status != "Running" {
			add("observer:off", "Continue without monitor (optional)", "")
			add("service:install", "Review monitor installation", "")
		}
	case "Models":
		add("enabled", checkbox(d.Enabled)+" Manage client", why)
		for _, id := range []string{"gpt-6.1-sol", "glm-5.3", "glm-5.3-flash"} {
			add("model:"+id, choice(d.Profile.Model == id)+" Model "+id, why)
		}
		for i, info := range m.models {
			add(fmt.Sprintf("discovered:%d", i), "Advertised model "+info.ID+" | "+info.Source, why)
		}
		for _, b := range []int64{272000, 353000, 500000} {
			add(fmt.Sprintf("budget:%d", b), choice(d.Profile.Budget == b)+fmt.Sprintf(" Budget %d", b), why)
		}
		for _, tier := range []string{"Standard", "Fast", "Ultrafast"} {
			add("tier:"+tier, choice(d.Profile.Tier == tier)+" Default tier "+tier, why)
		}
		add("capacity", checkbox(d.Profile.CapacityConfirmed)+" Confirm route supports this context budget", why)
		add("tiers", checkbox(d.Profile.TiersConfirmed)+" This route supports Fast and Ultrafast", why)
		for _, f := range []string{"model", "budget", "compact", "endpoint", "credential"} {
			add("advanced:"+f, "Advanced: edit "+f, why)
		}
		add("discover", "Discover advertised models", why)
		add("plan", "Review client draft", why)
		add("discard", "Discard this client's draft", "")
		add("back", "Back; keep draft", "")
	case "Review":
		add("review:back", "Back; keep draft", "")
		if m.plan != nil {
			add("review:read", "Next page / acknowledge last", "")
			reason := ""
			if !m.reviewRead {
				reason = "Read and acknowledge every page first"
			}
			if m.plan.Noop {
				reason = "No changes; nothing to apply"
			}
			add("review:apply", "Apply exact reviewed plan", reason)
		}
	case "Jobs":
		add("cancel", "Request cancellation", idleReason(m))
		add("back", "Back to dashboard", "")
	case "Gateway":
		add("advanced:endpoint", "Edit local endpoint", why)
		add("key", "Enter API key (hidden)", "")
		add("back", "Back", "")
	case "Service":
		for _, a := range []string{"status", "start", "stop", "restart", "install", "uninstall"} {
			add("service:"+a, strings.Title(a)+" monitor", "")
		}
		add("back", "Back", "")
	case "Repair":
		add("repair", "Prepare redacted manual brief", "")
		add("repair:export", "Export brief to terminal scrollback", repairReason(m))
		add("back", "Back", "")
	case "SSH":
		add("ssh", "Hand off to SSH command", "")
		add("back", "Back", "")
	case "Utilities":
		add("unavailable", "Install / update unavailable", "No reviewed utility installer or updater is provided by this backend")
		add("back", "Back", "")
	case "Settings":
		add("theme", "Theme: "+m.options.Theme, "")
		add("ascii", checkbox(m.options.ASCII)+" ASCII decoration", "")
		add("color", checkbox(m.options.NoColor)+" No color", "")
		add("motion", checkbox(m.options.ReducedMotion)+" Reduced motion", "")
		add("key", "Enter API key (session only by default)", "")
		add("nav:Utilities", "Utilities", "")
		add("nav:Service", "Background monitor", "")
		add("back", "Back", "")
	}
	add("help", "Help (?)", "")
	add("palette", "Commands (Ctrl+P)", "")
	return cs
}
func checkbox(b bool) string {
	if b {
		return "[x]"
	}
	return "[ ]"
}
func choice(b bool) string {
	if b {
		return "(o)"
	}
	return "( )"
}
func noClient(m *model) string {
	if _, ok := m.client(); !ok {
		return "No client selected"
	}
	return ""
}
func idleReason(m *model) string {
	if m.busy == "" {
		return "No running action"
	}
	return ""
}
func repairReason(m *model) string {
	if m.repair == "" {
		return "Prepare the brief first"
	}
	return ""
}
func (m *model) matches(id, name string) bool {
	return strings.Contains(strings.ToLower(id+" "+name), strings.ToLower(m.filter))
}
