package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/clankercode/oneline-ccc/internal/core"
)

const inventoryLimit int64 = 5 * 1024 * 1024

func validID(id string) bool {
	return id != "" && len(id) <= 256 && utf8.ValidString(id) && strings.IndexFunc(id, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}

// Discover performs one explicit GET to the endpoint's /models resource. An
// inventory entry proves availability only; response capacity fields are ignored.
func Discover(ctx context.Context, baseURL, key string) ([]core.ModelInfo, error) {
	return discover(ctx, baseURL, key, nil)
}
func discover(ctx context.Context, baseURL, key string, client *http.Client) ([]core.ModelInfo, error) {
	if core.ValidateEndpoint(baseURL) != nil {
		return nil, &Error{Kind: "endpoint"}
	}
	if !utf8.ValidString(key) || len(key) > 2560 || strings.IndexFunc(key, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return nil, &Error{Kind: "credential"}
	}
	data, err := get(ctx, strings.TrimRight(baseURL, "/")+"/models", key, inventoryLimit, client)
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(data, &root) != nil || root == nil {
		return nil, &Error{Kind: "inventory"}
	}
	var inventory []map[string]json.RawMessage
	if json.Unmarshal(root["data"], &inventory) != nil || inventory == nil {
		return nil, &Error{Kind: "inventory"}
	}
	out := make([]core.ModelInfo, 0, len(inventory))
	seen := map[string]bool{}
	for _, item := range inventory {
		var id string
		if json.Unmarshal(item["id"], &id) != nil || !validID(id) || seen[id] {
			return nil, &Error{Kind: "inventory"}
		}
		seen[id] = true
		out = append(out, core.ModelInfo{ID: id, Source: "gateway inventory (availability only; capacity unknown)"})
	}
	return out, nil
}
