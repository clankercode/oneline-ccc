// Package codex stages native configuration changes without rewriting foreign TOML.
package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/pelletier/go-toml/v2"
)

const SupportedVersion = "0.166.0"
const begin = "# BEGIN oneline-ccc LLMP managed configuration"
const end = "# END oneline-ccc LLMP managed configuration"

type Adapter struct {
	Home, Scratch string
	Program       codexrpc.Program
}

func New(home, scratch, explicit string) *Adapter {
	p, _ := codexrpc.FindProgram(explicit)
	return &Adapter{Home: home, Scratch: scratch, Program: p}
}
func (a *Adapter) Inspect(ctx context.Context) (core.Client, error) {
	c := core.Client{ID: "codex", Name: "Codex", Profile: core.Defaults(), Status: "NOT INSTALLED"}
	if a.Program.Path == "" {
		c.Reason = "Native Codex executable not found; wrappers are not executed during detection."
		return c, nil
	}
	c.Installed = true
	c.Executable = a.Program.Path
	stage, err := a.stage()
	if err != nil {
		return c, err
	}
	defer os.RemoveAll(stage)
	c.Version, err = codexrpc.Version(ctx, a.Program, stage)
	if err != nil {
		c.Status = "UNVERIFIED"
		c.Reason = "Cannot verify the selected native Codex executable."
		return c, nil
	}
	if c.Version != SupportedVersion {
		c.Status = "READ ONLY"
		c.Reason = "This adapter is verified for Codex " + SupportedVersion + "; this version needs a compatibility check."
		return c, nil
	}
	file, err := a.readConfig("config.toml")
	if err != nil {
		return c, errors.New("Cannot inspect Codex configuration safely")
	}
	var values map[string]any
	if len(file.Before) > 0 && toml.Unmarshal(file.Before, &values) != nil {
		c.Status = "READ ONLY"
		c.Reason = "Existing TOML is invalid; preserve it and correct it before applying changes."
		return c, nil
	}
	if hasInlineSecret(values) {
		c.Status = "READ ONLY"
		c.Reason = "Inline credentials require manual migration to environment references before managed changes."
		return c, nil
	}
	managedProvider := table(table(values["model_providers"])["oneline_llmp"])
	if _, ok := managedProvider["env_http_headers"]; ok {
		c.Status = "READ ONLY"
		c.Reason = "Existing managed-provider header authentication requires manual migration before changes."
		return c, nil
	}
	c.Supported = true
	c.Status = "NEEDS SETUP"
	provider := table(table(values["model_providers"])[text(values["model_provider"])])
	p := core.Defaults()
	p.Model = first(text(values["model"]), p.Model)
	p.BaseURL = text(provider["base_url"])
	p.EnvKey = first(text(provider["env_key"]), p.EnvKey)
	if n, ok := values["model_context_window"].(int64); ok && n > 0 {
		p.Budget = n
		p.CapacityConfirmed = true
	}
	if n, ok := values["model_auto_compact_token_limit"].(int64); ok && n > 0 {
		p.Compact = n
	} else {
		p.Compact = p.Budget * 9 / 10
	}
	p.Tier = displayTier(text(values["service_tier"]))
	p.Catalog = text(values["model_catalog_json"])
	features := table(values["features"])
	p.TiersConfirmed = features["ultrafast_mode"] == true && text(values["model_provider"]) == "oneline_llmp"
	if p.BaseURL != "" {
		c.Status = "CONFIGURED"
	}
	c.Profile = p
	return c, nil
}

// readConfig can plan a first setup without creating the client directory.
// Apply revalidates the snapshot after explicitly creating the private home.
func (a *Adapter) readConfig(name string) (configtxn.Write, error) {
	path, err := filepath.Abs(filepath.Join(a.Home, name))
	if err != nil {
		return configtxn.Write{}, configtxn.ErrUnsafe
	}
	_, err = os.Lstat(filepath.Dir(path))
	if err == nil {
		return configtxn.Read(path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return configtxn.Write{}, configtxn.ErrUnsafe
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		info, e := os.Lstat(dir)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return configtxn.Write{}, configtxn.ErrUnsafe
		}
		if e == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return configtxn.Write{}, configtxn.ErrUnsafe
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return configtxn.Write{Path: path, Mode: 0600}, nil
}
func (a *Adapter) stage() (string, error) {
	if err := configtxn.PrivateDir(a.Scratch); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(a.Scratch, "codex-stage-")
	if err != nil {
		return "", errors.New("Cannot create private Codex staging directory")
	}
	if err = configtxn.PrivateDir(stage); err != nil {
		_ = os.RemoveAll(stage)
		return "", err
	}
	return stage, nil
}
func table(value any) map[string]any {
	m, _ := value.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func text(value any) string { s, _ := value.(string); return s }
func first(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
func displayTier(value string) string {
	switch value {
	case "", "default":
		return "Standard"
	case "fast", "priority":
		return "Fast"
	case "ultrafast":
		return "Ultrafast"
	}
	return value
}
func wireTier(value string) string {
	switch value {
	case "", "Standard", "default":
		return "default"
	case "Fast", "fast", "priority":
		return "fast"
	case "Ultrafast", "ultrafast":
		return "ultrafast"
	}
	return value
}
func hasInlineSecret(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			k := strings.ToLower(key)
			if k == "api_key" || strings.HasSuffix(k, "_api_key") || k == "experimental_bearer_token" || k == "http_headers" || k == "authorization" || k == "password" {
				return true
			}
			if hasInlineSecret(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasInlineSecret(child) {
				return true
			}
		}
	}
	return false
}
