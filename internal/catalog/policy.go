package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
)

const nativeLimit int64 = 1024 * 1024

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

// ConfirmedCodex downloads the exact installed version's official native
// catalog. It preserves the full matching model and root metadata, and modifies
// only confirmed tier fields. Callers must separately confirm route capacity.
func ConfirmedCodex(ctx context.Context, version, model string, tiers []string) ([]byte, error) {
	return confirmedCodex(ctx, version, model, tiers, nil)
}
func confirmedCodex(ctx context.Context, version, model string, tiers []string, client *http.Client) ([]byte, error) {
	if len(version) > 128 || !versionPattern.MatchString(version) || (model != "gpt-6.1-sol" && model != "gpt-6-astra") {
		return nil, &Error{Kind: "preset"}
	}
	normalized := []string{}
	seen := map[string]bool{}
	for _, tier := range tiers {
		switch tier {
		case "default":
			continue
		case "fast":
			tier = "priority"
		case "priority", "ultrafast":
		default:
			return nil, &Error{Kind: "preset"}
		}
		if !seen[tier] {
			normalized = append(normalized, tier)
			seen[tier] = true
		}
	}
	data, err := get(ctx, "https://raw.githubusercontent.com/openai/codex/rust-v"+version+"/codex-rs/models-manager/models.json", "", nativeLimit, client)
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(data, &root) != nil || root == nil {
		return nil, &Error{Kind: "native"}
	}
	var models []map[string]json.RawMessage
	if json.Unmarshal(root["models"], &models) != nil || models == nil {
		return nil, &Error{Kind: "native"}
	}
	var selected map[string]json.RawMessage
	modelIDs := map[string]bool{}
	for _, entry := range models {
		var slug string
		if json.Unmarshal(entry["slug"], &slug) != nil || !validID(slug) || modelIDs[slug] {
			return nil, &Error{Kind: "native"}
		}
		modelIDs[slug] = true
		if slug == model {
			if selected != nil {
				return nil, &Error{Kind: "native"}
			}
			selected = entry
		}
	}
	if selected == nil {
		return nil, &Error{Kind: "native"}
	}
	var visibility string
	var supported bool
	var upstream []map[string]json.RawMessage
	if json.Unmarshal(selected["visibility"], &visibility) != nil || visibility != "list" || json.Unmarshal(selected["supported_in_api"], &supported) != nil || !supported || json.Unmarshal(selected["service_tiers"], &upstream) != nil || upstream == nil {
		return nil, &Error{Kind: "native"}
	}
	nativeTiers := map[string]map[string]json.RawMessage{}
	for _, tier := range upstream {
		var id, name, description string
		if json.Unmarshal(tier["id"], &id) != nil || id == "" || nativeTiers[id] != nil || !stringField(tier["name"], &name) || !stringField(tier["description"], &description) {
			return nil, &Error{Kind: "native"}
		}
		nativeTiers[id] = tier
	}
	serviceTiers := []map[string]json.RawMessage{}
	legacy := []string{}
	for _, tier := range normalized {
		if tier == "priority" {
			if nativeTiers[tier] == nil {
				return nil, &Error{Kind: "native"}
			}
			serviceTiers = append(serviceTiers, nativeTiers[tier])
			legacy = append(legacy, "fast")
		} else {
			serviceTiers = append(serviceTiers, map[string]json.RawMessage{"id": json.RawMessage(`"ultrafast"`), "name": json.RawMessage(`"Ultrafast"`), "description": json.RawMessage(`"Lower latency, increased usage; requires an eligible plan"`)})
			legacy = append(legacy, "ultrafast")
		}
	}
	selected["service_tiers"], _ = json.Marshal(serviceTiers)
	selected["additional_speed_tiers"], _ = json.Marshal(legacy)
	selected["default_service_tier"] = json.RawMessage("null")
	root["models"], _ = json.Marshal([]map[string]json.RawMessage{selected})
	out, err := json.Marshal(root)
	if err != nil {
		return nil, &Error{Kind: "native"}
	}
	return out, nil
}
func stringField(raw json.RawMessage, out *string) bool {
	return len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, out) == nil
}
