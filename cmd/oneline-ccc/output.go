package main

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

var urls = regexp.MustCompile(`(?i)https?://[^\s]+`)
var credentialText = regexp.MustCompile(`(?i)(bearer\s+[^\s]+|\bsk-[A-Za-z0-9_-]+)`)

func cleanText(s string) string {
	s = ansi.Strip(s)
	s = strings.Map(func(r rune) rune {
		if r != '\n' && (unicode.IsControl(r) || unicode.Is(unicode.Cf, r)) {
			return -1
		}
		return r
	}, s)
	return s
}
func safe(s string) string {
	return credentialText.ReplaceAllString(urls.ReplaceAllString(cleanText(s), "[private route]"), "[credential hidden]")
}
func publicSnapshot(s core.Snapshot) core.Snapshot {
	// Copy slices before redaction so a reusable backend snapshot is unchanged.
	s.Clients = append([]core.Client(nil), s.Clients...)
	s.Receipts = append([]core.Receipt(nil), s.Receipts...)
	var private []string
	for _, c := range s.Clients {
		for _, value := range []string{c.Executable, c.Profile.BaseURL, c.Profile.Catalog} {
			if value != "" {
				private = append(private, value)
			}
		}
	}
	sort.Slice(private, func(i, j int) bool { return len(private[i]) > len(private[j]) })
	exact := func(value string) string {
		for _, p := range private {
			value = strings.ReplaceAll(value, p, "[private]")
		}
		return cleanText(value)
	}
	scrub := func(value string) string {
		for _, p := range private {
			value = strings.ReplaceAll(value, p, "[private]")
		}
		return safe(value)
	}
	for i := range s.Clients {
		c := &s.Clients[i]
		c.ID = exact(c.ID)
		c.Name = scrub(c.Name)
		c.Status = scrub(c.Status)
		c.Reason = scrub(c.Reason)
		c.Version = scrub(c.Version)
		c.Executable = ""
		c.Profile.BaseURL = ""
		c.Profile.Catalog = ""
		c.Profile.Model = exact(c.Profile.Model)
		c.Profile.Tier = scrub(c.Profile.Tier)
		c.Profile.EnvKey = exact(c.Profile.EnvKey)
	}
	for i := range s.Receipts {
		r := &s.Receipts[i]
		r.ID = exact(r.ID)
		r.ClientID = exact(r.ClientID)
		r.Outcome = scrub(r.Outcome)
		r.Summary = scrub(r.Summary)
	}
	switch s.Credential {
	case "Missing", "Locked", "Stored", "Session only":
	default:
		s.Credential = "Unknown"
	}
	s.Service.Backend = scrub(s.Service.Backend)
	s.Service.Status = scrub(s.Service.Status)
	s.Service.Detail = scrub(s.Service.Detail)
	return s
}
func printStatus(w io.Writer, s core.Snapshot, jsonOutput, doctor bool) error {
	s = publicSnapshot(s)
	if jsonOutput {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(s)
	}
	if doctor {
		fmt.Fprintln(w, "Native Codex readiness; no authentication or inference check")
	}
	fmt.Fprintln(w, "oneline-ccc | Local inspection | Gateway A")
	for _, c := range s.Clients {
		fmt.Fprintf(w, "%s (%s): %s\n  Installed: %t | Supported: %t | Managed: %t\n  Model: %s | Budget: %d | Default tier: %s\n", c.Name, c.ID, c.Status, c.Installed, c.Supported, c.Managed, c.Profile.Model, c.Profile.Budget, c.Profile.Tier)
		if c.Reason != "" {
			fmt.Fprintln(w, "  Reason: "+c.Reason)
		}
	}
	if len(s.Clients) == 0 {
		fmt.Fprintln(w, "No clients discovered")
	}
	fmt.Fprintf(w, "Credential: %s\nMonitor: %s (%s)\n", s.Credential, s.Service.Status, s.Service.Backend)
	fmt.Fprintln(w, "Verified native configuration path: Codex only. Inference: not verified.")
	_, err := fmt.Fprintln(w, "TUI: use a terminal; Tab/Enter/Space, ? help, Ctrl+P commands.\nSource build: go build -o oneline-ccc ./cmd/oneline-ccc")
	return err
}
