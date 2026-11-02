package monitor

import (
	"bytes"
	"encoding/xml"
	"path/filepath"
	"regexp"
	"strings"
)

const unitName = "oneline-ccc-observer.service"
const agentLabel = "com.oneline-ccc.observer"

type serviceSpec struct {
	backend, artifact, target string
	definition                []byte
}

func (m *Manager) backend() string {
	switch m.os {
	case "linux":
		return "systemd user"
	case "darwin":
		return "LaunchAgent"
	case "windows":
		return "Scheduled Task (current user logon)"
	default:
		return "user-session observer"
	}
}
func systemdQuote(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`, `$`, `$$`).Replace(value) + `"`
}
func xmlText(value string) string {
	var out bytes.Buffer
	xml.EscapeText(&out, []byte(value))
	return out.String()
}
func (m *Manager) spec() (serviceSpec, error) {
	spec := serviceSpec{backend: m.backend()}
	if !m.validRoots() || !safePath(m.home) || !safePath(m.executable) {
		return spec, errUnsafe
	}
	switch m.os {
	case "linux":
		spec.artifact = filepath.Join(m.home, ".config", "systemd", "user", unitName)
		spec.target = unitName
		quoted := []string{}
		for _, arg := range m.argv() {
			quoted = append(quoted, systemdQuote(arg))
		}
		spec.definition = []byte("# oneline-ccc read-only observer v1\n[Unit]\nDescription=oneline-ccc read-only observer\n[Service]\nType=simple\nExecStart=" + strings.Join(quoted, " ") + "\nRestart=no\nUMask=0077\nNoNewPrivileges=yes\nStandardOutput=null\nStandardError=null\n[Install]\nWantedBy=default.target\n")
	case "darwin":
		if !regexp.MustCompile(`^[0-9]+$`).MatchString(m.uid) {
			return spec, errUnavailable
		}
		spec.artifact = filepath.Join(m.home, "Library", "LaunchAgents", agentLabel+".plist")
		spec.target = "gui/" + m.uid + "/" + agentLabel
		var args strings.Builder
		for _, arg := range m.argv() {
			args.WriteString("<string>" + xmlText(arg) + "</string>")
		}
		spec.definition = []byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>` + agentLabel + `</string><key>ProgramArguments</key><array>` + args.String() + `</array><key>RunAtLoad</key><true/><key>KeepAlive</key><false/><key>ProcessType</key><string>Background</string><key>Umask</key><integer>63</integer><key>StandardOutPath</key><string>/dev/null</string><key>StandardErrorPath</key><string>/dev/null</string></dict></plist>`)
	default:
		return spec, errUnavailable
	}
	if len(spec.definition) > recordLimit {
		return spec, errUnsafe
	}
	return spec, nil
}
