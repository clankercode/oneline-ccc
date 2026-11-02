package catalog

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/clankercode/oneline-ccc/internal/core"
)

// models.dev's multi-provider document is larger than a gateway inventory. The
// explicit 32 MiB cap bounds both uncompressed input and parsing work.
const metadataLimit = 32 * 1024 * 1024
const metadataURL = "https://models.dev/api.json"

type capacity struct {
	value int64
	known bool
}

// Enrich consults a fixed public URL, never a gateway or credential supplied by
// the caller. Exact inventory keys and explicit canonical IDs can contribute;
// differing capacities or incomplete matches remain unknown. Context is public
// model metadata only, not confirmed route capacity or account entitlement.
func Enrich(ctx context.Context, models []core.ModelInfo) ([]core.ModelInfo, error) {
	return enrich(ctx, models, nil)
}
func enrich(ctx context.Context, models []core.ModelInfo, client *http.Client) ([]core.ModelInfo, error) {
	wanted := map[string]bool{}
	for _, model := range models {
		if !validID(model.ID) || wanted[model.ID] {
			return nil, &Error{Kind: "inventory"}
		}
		wanted[model.ID] = true
	}
	out := append([]core.ModelInfo{}, models...)
	if len(out) == 0 {
		return out, nil
	}
	data, err := get(ctx, metadataURL, "", metadataLimit, client)
	if err != nil {
		return nil, err
	}
	var providers map[string]json.RawMessage
	if json.Unmarshal(data, &providers) != nil || providers == nil {
		return nil, &Error{Kind: "metadata"}
	}
	matches := map[string]capacity{}
	add := func(id string, c capacity) {
		if !wanted[id] {
			return
		}
		if previous, ok := matches[id]; ok {
			if !previous.known || !c.known || previous.value != c.value {
				matches[id] = capacity{}
			}
			return
		}
		matches[id] = c
	}
	for _, raw := range providers {
		var provider map[string]json.RawMessage
		if json.Unmarshal(raw, &provider) != nil || provider == nil {
			return nil, &Error{Kind: "metadata"}
		}
		var entries map[string]json.RawMessage
		if json.Unmarshal(provider["models"], &entries) != nil || entries == nil {
			return nil, &Error{Kind: "metadata"}
		}
		for id, rawModel := range entries {
			var fields map[string]json.RawMessage
			if json.Unmarshal(rawModel, &fields) != nil || fields == nil {
				add(id, capacity{})
				continue
			}
			var canonical string
			if value, exists := fields["id"]; exists && (json.Unmarshal(value, &canonical) != nil || !validID(canonical)) {
				add(id, capacity{})
				continue
			}
			var limits map[string]json.RawMessage
			var contextLimit int64
			c := capacity{}
			if json.Unmarshal(fields["limit"], &limits) == nil && json.Unmarshal(limits["context"], &contextLimit) == nil && contextLimit > 0 {
				c = capacity{value: contextLimit, known: true}
			}
			add(id, c)
			if canonical != id && validID(canonical) {
				add(canonical, c)
			}
		}
	}
	for i := range out {
		out[i].Context = 0
		out[i].Source = "gateway inventory (availability only; capacity unknown)"
		if c, ok := matches[out[i].ID]; ok && c.known {
			out[i].Context = c.value
			out[i].Source = "models.dev (public model metadata; route capacity and entitlement unverified)"
		}
	}
	return out, nil
}
