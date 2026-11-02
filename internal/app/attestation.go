package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/pelletier/go-toml/v2"
)

// Only domain-separated digests enter settings; the private tuple stays in memory.
type attestation struct {
	Capacity string `json:"capacity,omitempty"`
	Tiers    string `json:"tiers,omitempty"`
}

func fingerprint(p core.Profile, version, scope string) string {
	// Profile's public JSON deliberately hides BaseURL and Catalog. Explicitly
	// include both here so route/catalog drift can never reuse confirmation.
	data, _ := json.Marshal(struct {
		Scope         string `json:"scope"`
		BaseURL       string `json:"base_url"`
		Model         string `json:"model"`
		Budget        int64  `json:"budget"`
		Compact       int64  `json:"compact"`
		Tier          string `json:"tier"`
		EnvKey        string `json:"env_key"`
		Catalog       string `json:"catalog"`
		NativeVersion string `json:"native_version"`
	}{scope, p.BaseURL, p.Model, p.Budget, p.Compact, canonicalTier(p.Tier), p.EnvKey, p.Catalog, version})
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func evidence(p core.Profile, version string) attestation {
	var out attestation
	if p.CapacityConfirmed {
		out.Capacity = fingerprint(p, version, "capacity-v1")
	}
	if p.TiersConfirmed {
		out.Tiers = fingerprint(p, version, "tiers-v1")
	}
	return out
}

func restoreAttestation(p core.Profile, version string, managed bool, a attestation) core.Profile {
	p.CapacityConfirmed = managed && a.Capacity != "" && a.Capacity == fingerprint(p, version, "capacity-v1")
	p.TiersConfirmed = managed && a.Tiers != "" && a.Tiers == fingerprint(p, version, "tiers-v1")
	return p
}

// The adapter can select a content-addressed catalog path during staging. Bind
// the actual validated main configuration, rather than the caller's path hint.
func stagedCatalog(home string, writes []configtxn.Write) (string, error) {
	for _, w := range writes {
		if w.Path != filepath.Join(home, "config.toml") {
			continue
		}
		var values map[string]any
		if toml.Unmarshal(w.Data, &values) != nil {
			return "", errors.New("native adapter staged invalid configuration")
		}
		value, exists := values["model_catalog_json"]
		if !exists {
			return "", nil
		}
		path, ok := value.(string)
		if !ok {
			return "", errors.New("native adapter staged invalid catalog reference")
		}
		return path, nil
	}
	return "", errors.New("native adapter omitted the default configuration")
}
