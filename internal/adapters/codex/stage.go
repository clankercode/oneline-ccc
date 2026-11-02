package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/pelletier/go-toml/v2"
)

// Prepare returns reviewed bytes only; the transaction engine owns real writes.
func (a *Adapter) Prepare(ctx context.Context, p core.Profile, catalog []byte) ([]configtxn.Write, error) {
	if err := core.ValidateProfile(p); err != nil {
		return nil, err
	}
	c, err := a.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	if !c.Supported {
		return nil, errors.New(c.Reason)
	}
	main, err := a.readConfig("config.toml")
	if err != nil {
		return nil, err
	}
	profile, err := a.readConfig("llmp.config.toml")
	if err != nil {
		return nil, err
	}
	if profile.Existed && !ownedProfile(profile.Before) {
		return nil, errors.New("Existing llmp profile is not exclusively manager-owned; preserve or rename it before applying")
	}
	var writes []configtxn.Write
	if len(catalog) > 0 {
		hash := sha256.Sum256(catalog)
		cache := filepath.Join(a.Scratch, "catalogs")
		if e := configtxn.PrivateDir(cache); e != nil {
			return nil, e
		}
		p.Catalog = filepath.Join(cache, "models-"+hex.EncodeToString(hash[:])+".json")
		file, e := configtxn.Read(p.Catalog)
		if e != nil {
			return nil, e
		}
		if file.Existed && !reflect.DeepEqual(file.Before, catalog) {
			return nil, errors.New("Immutable model catalog has changed; preserve and inspect it")
		}
		if !file.Existed {
			if e = configtxn.PrivateFile(p.Catalog, catalog); e != nil {
				return nil, e
			}
		}
	}
	if p.TiersConfirmed && p.Catalog == "" {
		return nil, errors.New("Confirmed accelerated tiers require a complete version-matched model catalog")
	}
	if p.Catalog != "" && !filepath.IsAbs(p.Catalog) {
		return nil, errors.New("Choose an absolute model catalog path")
	}
	// Metadata is a private immutable app cache, not a client configuration write.
	// Its stable path lets native validation read exactly the reviewed catalog.
	main.Data, err = a.edit(ctx, main.Before, settings(p))
	if err != nil {
		return nil, err
	}
	profile.Data = []byte(renderProfile(p))
	if _, err = a.edit(ctx, profile.Data, settings(p)); err != nil {
		return nil, err
	}
	return append(writes, main, profile), nil
}
func settings(p core.Profile) map[string]any {
	values := map[string]any{"model": p.Model, "model_provider": "oneline_llmp", "model_context_window": p.Budget, "model_auto_compact_token_limit": p.Compact, "service_tier": wireTier(p.Tier), "model_catalog_json": nil, "features.ultrafast_mode": nil,
		"model_providers.oneline_llmp.name": "LLMP (oneline-ccc)", "model_providers.oneline_llmp.base_url": p.BaseURL, "model_providers.oneline_llmp.wire_api": "responses", "model_providers.oneline_llmp.env_key": p.EnvKey, "model_providers.oneline_llmp.requires_openai_auth": false}
	if p.Catalog != "" {
		values["model_catalog_json"] = p.Catalog
	}
	if p.TiersConfirmed {
		values["features.ultrafast_mode"] = true
	}
	return values
}
func (a *Adapter) edit(ctx context.Context, before []byte, values map[string]any) ([]byte, error) {
	stage, err := a.stage()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	path := filepath.Join(stage, "config.toml")
	if err = configtxn.PrivateFile(path, before); err != nil {
		return nil, err
	}
	err = codexrpc.Run(ctx, a.Program, stage, func(c *codexrpc.Client) error {
		var read struct {
			Layers []struct {
				Name    struct{ Type, File, Profile string }
				Version string
			}
		}
		if e := c.Call("config/read", map[string]any{"includeLayers": true}, &read); e != nil {
			return e
		}
		version := ""
		for _, layer := range read.Layers {
			if layer.Name.Type == "user" && layer.Name.Profile == "" && filepath.Clean(layer.Name.File) == path {
				if version != "" {
					return errors.New("Ambiguous native user configuration layer")
				}
				version = layer.Version
			}
		}
		if version == "" {
			return errors.New("Native writable user layer not found")
		}
		edits := []map[string]any{}
		for key, value := range values {
			edits = append(edits, map[string]any{"keyPath": key, "value": value, "mergeStrategy": "replace"})
		}
		var result struct {
			Status, FilePath   string
			OverriddenMetadata json.RawMessage
		}
		if e := c.Call("config/batchWrite", map[string]any{"filePath": path, "expectedVersion": version, "edits": edits, "reloadUserConfig": false}, &result); e != nil {
			return e
		}
		if result.Status != "ok" || filepath.Clean(result.FilePath) != path || len(result.OverriddenMetadata) > 0 && string(result.OverriddenMetadata) != "null" {
			return errors.New("Native configuration write was overridden")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = codexrpc.Run(ctx, a.Program, stage, func(c *codexrpc.Client) error {
		var read struct{ Config map[string]any }
		if e := c.Call("config/read", map[string]any{"includeLayers": false}, &read); e != nil {
			return e
		}
		for key, expected := range values {
			var actual any = read.Config
			for _, part := range strings.Split(key, ".") {
				actual = table(actual)[part]
			}
			left, _ := json.Marshal(actual)
			right, _ := json.Marshal(expected)
			if string(left) != string(right) {
				return errors.New("Fresh Codex process did not select the reviewed settings")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if catalogPath := text(values["model_catalog_json"]); catalogPath != "" {
		if err = a.validateCatalog(ctx, stage, text(values["model"]), values["features.ultrafast_mode"] == true); err != nil {
			return nil, err
		}
	}
	return os.ReadFile(path)
}
func quote(value string) string { data, _ := json.Marshal(value); return string(data) }
func renderProfile(p core.Profile) string {
	value := begin + "\nmodel_provider = \"oneline_llmp\"\nmodel = " + quote(p.Model) + "\nmodel_context_window = " + number(p.Budget) + "\nmodel_auto_compact_token_limit = " + number(p.Compact) + "\nservice_tier = " + quote(wireTier(p.Tier)) + "\n"
	if p.Catalog != "" {
		value += "model_catalog_json = " + quote(p.Catalog) + "\n"
	}
	if p.TiersConfirmed {
		value += "features.ultrafast_mode = true\n"
	}
	return value + "\n[model_providers.oneline_llmp]\nname = \"LLMP (oneline-ccc)\"\nbase_url = " + quote(p.BaseURL) + "\nwire_api = \"responses\"\nenv_key = " + quote(p.EnvKey) + "\nrequires_openai_auth = false\n\n" + end + "\n"
}
func number(n int64) string { data, _ := json.Marshal(n); return string(data) }
func ownedProfile(data []byte) bool {
	text := strings.TrimSpace(string(data))
	if !strings.HasPrefix(text, begin+"\n") && !strings.HasPrefix(text, begin+"\r\n") {
		return false
	}
	if !strings.HasSuffix(text, end) {
		return false
	}
	var doc map[string]any
	if toml.Unmarshal(data, &doc) != nil || hasInlineSecret(doc) {
		return false
	}
	for key := range doc {
		switch key {
		case "model", "model_provider", "model_context_window", "model_auto_compact_token_limit", "service_tier", "model_catalog_json", "features", "model_providers":
		default:
			return false
		}
	}
	if raw, exists := doc["features"]; exists {
		if _, ok := raw.(map[string]any); !ok {
			return false
		}
	}
	for key := range table(doc["features"]) {
		if key != "ultrafast_mode" {
			return false
		}
	}
	providers := table(doc["model_providers"])
	if len(providers) != 1 {
		return false
	}
	provider, ok := providers["oneline_llmp"].(map[string]any)
	if !ok {
		return false
	}
	for key := range provider {
		switch key {
		case "name", "base_url", "wire_api", "env_key", "requires_openai_auth":
		default:
			return false
		}
	}
	return textValue(doc["model_provider"]) == "oneline_llmp"
}
func textValue(value any) string { return text(value) }
