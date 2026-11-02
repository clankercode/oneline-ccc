package tui

import "github.com/clankercode/oneline-ccc/internal/core"

func (m *model) sshReason() string {
	if availability, ok := m.backend.(core.SSHSetupAvailability); ok {
		return m.public(availability.SSHSetupUnavailableReason())
	}
	return "SSH setup handoff is unavailable for this backend"
}

func (m *model) monitorReason(action string) string {
	if action == "status" {
		return ""
	}
	switch m.snapshot.Service.Status {
	case "Not installed":
		if action == "install" {
			return ""
		}
		return "Install the optional monitor first; inspect and review installation"
	case "Running", "Stopped", "Failed":
		if action == "install" {
			return "Monitor is already installed; inspect its current state"
		}
		if action == "start" && m.snapshot.Service.Status == "Running" {
			return "Monitor is already running"
		}
		if action == "stop" && m.snapshot.Service.Status == "Stopped" {
			return "Monitor is already stopped"
		}
		return ""
	default:
		return "Inspect monitor status and resolve its state before requesting a change"
	}
}
