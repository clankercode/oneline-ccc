package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clankercode/oneline-ccc/internal/core"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixtureClient(t *testing.T, target string, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	transport := server.Client().Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != target || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Referer") != "" {
			t.Error("public request carried private data or wrong URL")
			return nil, errors.New("fixture request refused")
		}
		clone := r.Clone(r.Context())
		u := *r.URL
		clone.URL = &u
		clone.URL.Scheme = "http"
		clone.URL.Host = strings.TrimPrefix(server.URL, "http://")
		return transport.RoundTrip(clone)
	})}
}
func TestDiscoverExactInventoryNoCapacity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Error("incorrect inventory request")
		}
		io.WriteString(w, `{"data":[{"id":"Exact/Model","context_window":999999},{"id":"alias--fast"}]}`)
	}))
	defer server.Close()
	models, err := Discover(context.Background(), server.URL+"/v1/", "fixture-key")
	if err != nil || len(models) != 2 || models[0].ID != "Exact/Model" || models[1].ID != "alias--fast" || models[0].Context != 0 || !strings.Contains(models[0].Source, "availability") {
		t.Fatalf("inventory failed: %v", err)
	}
}
func TestDiscoverBoundsAndSafeErrors(t *testing.T) {
	tests := []struct {
		name, body string
		code       int
	}{
		{"unauthorized", "fixture-private-body", 401}, {"forbidden", "fixture-private-body", 403}, {"rate", "fixture-private-body", 429},
		{"invalid", "not JSON fixture-private-body", 200}, {"empty", "{}", 200}, {"null", `{"data":null}`, 200},
		{"trailing", `{"data":[]} {}`, 200}, {"duplicate ID", `{"data":[{"id":"x"},{"id":"x"}]}`, 200},
		{"duplicate field", `{"data":[{"id":"x","id":"y"}]}`, 200}, {"whitespace ID", `{"data":[{"id":"bad id"}]}`, 200},
		{"control ID", `{"data":[{"id":"bad\u0000id"}]}`, 200}, {"long ID", `{"data":[{"id":"` + strings.Repeat("x", 257) + `"}]}`, 200},
		{"oversize", strings.Repeat(" ", 5*1024*1024+1), 200}, {"invalid UTF8", "{\"data\":[{\"id\":\"\xff\"}]}", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.code); io.WriteString(w, tc.body) }))
			defer s.Close()
			_, err := Discover(context.Background(), s.URL, "fixture-key")
			if err == nil || strings.Contains(err.Error(), "fixture-") || strings.Contains(err.Error(), s.URL) || len(err.Error()) > 256 {
				t.Fatal("missing or unsafe bounded error")
			}
		})
	}
	for _, endpoint := range []string{"http://external.invalid/v1", "https://user:pass@private.invalid", "https://private.invalid?key=fixture-key"} {
		if _, err := Discover(context.Background(), endpoint, "fixture-key"); err == nil || strings.Contains(err.Error(), "fixture-key") {
			t.Fatal("bad endpoint accepted")
		}
	}
}
func TestRedirectNeverForwardsToken(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer origin.Close()
	if _, err := Discover(context.Background(), origin.URL, "fixture-key"); err == nil || reached {
		t.Fatal("redirect was followed")
	}
}
func TestTLSAndCancellationAndTimeoutPolicy(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"data":[]}`) }))
	defer s.Close()
	_, err := Discover(context.Background(), s.URL, "fixture-key")
	var requestErr *Error
	if !errors.As(err, &requestErr) || requestErr.Kind != "tls" {
		t.Fatal("TLS error not classified")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Discover(ctx, "http://localhost", "fixture-key"); err == nil {
		t.Fatal("cancellation ignored")
	}
	client := boundedClient(&http.Client{Timeout: time.Hour, CheckRedirect: func(*http.Request, []*http.Request) error { return nil }})
	if client.Timeout != 10*time.Second || client.CheckRedirect(nil, nil) != http.ErrUseLastResponse || client.Jar != nil {
		t.Fatal("network policy not enforced")
	}
}
func TestEnrichExactAndAmbiguous(t *testing.T) {
	body := `{"a":{"models":{"Exact":{"id":"Exact","limit":{"context":500000}},"conflict":{"limit":{"context":500000}},"mapped-key":{"id":"canonical","limit":{"context":272000}}}},"b":{"models":{"conflict":{"limit":{"context":353000}}}}}`
	client := fixtureClient(t, "https://models.dev/api.json", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
	input := []core.ModelInfo{{ID: "Exact"}, {ID: "alias", Context: 999999}, {ID: "conflict"}, {ID: "canonical"}, {ID: "exact"}}
	out, err := enrich(context.Background(), input, client)
	if err != nil || len(out) != 5 || out[0].Context != 500000 || out[1].Context != 0 || out[2].Context != 0 || out[3].Context != 272000 || out[4].Context != 0 || !strings.Contains(out[0].Source, "models.dev") || !strings.Contains(out[0].Source, "unverified") {
		t.Fatalf("bad metadata evidence: %v", err)
	}
	if input[1].Context != 999999 {
		t.Fatal("input mutated")
	}
}
func TestPublicSourcesRejectRedirectsAndOversize(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{{"redirect", "", 302}, {"oversize", strings.Repeat(" ", metadataLimit+1), 200}, {"invalid", "{", 200}, {"401", "", 401}, {"rate", "", 429}, {"wrong shape", "[]", 200}} {
		t.Run(tc.name, func(t *testing.T) {
			client := fixtureClient(t, "https://models.dev/api.json", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "https://private.invalid/fixture-key")
				w.WriteHeader(tc.code)
				io.WriteString(w, tc.body)
			})
			if _, err := enrich(context.Background(), []core.ModelInfo{{ID: "alias"}}, client); err == nil || strings.Contains(err.Error(), "private.invalid") {
				t.Fatal("public metadata response accepted or leaked")
			}
		})
	}
}

const nativeCatalog = `{"version":"fixture","unknown_root":{"retain":true},"models":[{"slug":"gpt-6.1-sol","visibility":"list","supported_in_api":true,"service_tiers":[{"id":"priority","name":"Fast","description":"native description","extra":17}],"additional_speed_tiers":["fast"],"default_service_tier":"priority","instructions":"complete native instructions","tools":[{"type":"native","nested":{"retain":true}}],"context_window":1048576,"unknown_model":{"retain":true}},{"slug":"other","service_tiers":[]}]}`

func TestConfirmedCatalogRetainsFullNativeMetadata(t *testing.T) {
	const target = "https://raw.githubusercontent.com/openai/codex/rust-v0.135.0/codex-rs/models-manager/models.json"
	client := fixtureClient(t, target, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, nativeCatalog) })
	out, err := confirmedCodex(context.Background(), "0.135.0", "gpt-6.1-sol", []string{"default", "fast", "priority", "ultrafast"}, client)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	json.Unmarshal(out, &root)
	var models []map[string]json.RawMessage
	json.Unmarshal(root["models"], &models)
	var original map[string]json.RawMessage
	json.Unmarshal([]byte(nativeCatalog), &original)
	var originals []map[string]json.RawMessage
	json.Unmarshal(original["models"], &originals)
	if len(models) != 1 {
		t.Fatal("unexpected model count")
	}
	for k, v := range originals[0] {
		if k != "service_tiers" && k != "additional_speed_tiers" && k != "default_service_tier" && string(models[0][k]) != string(v) {
			t.Fatalf("lost native field %s", k)
		}
	}
	if string(root["unknown_root"]) != string(original["unknown_root"]) || string(models[0]["default_service_tier"]) != "null" {
		t.Fatal("root metadata/default lost")
	}
	var tiers []map[string]any
	json.Unmarshal(models[0]["service_tiers"], &tiers)
	if len(tiers) != 2 || tiers[0]["id"] != "priority" || tiers[0]["extra"] != float64(17) || tiers[1]["id"] != "ultrafast" {
		t.Fatal("native tiers lost or invented")
	}
	if string(models[0]["additional_speed_tiers"]) != `["fast","ultrafast"]` {
		t.Fatal("wrong legacy mapping")
	}
}
func TestConfirmedCatalogRefusesUnknownDuplicateMalformed(t *testing.T) {
	for _, body := range []string{"{}", `{"models":[]}`, strings.Replace(nativeCatalog, `"visibility":"list"`, `"visibility":"hidden"`, 1), strings.Replace(nativeCatalog, `"supported_in_api":true`, `"supported_in_api":false`, 1), strings.Replace(nativeCatalog, `"slug":"other"`, `"slug":"gpt-6.1-sol"`, 1), strings.Replace(nativeCatalog, `"description":"native description",`, ``, 1), strings.Replace(nativeCatalog, `"slug":"gpt-6.1-sol"`, `"slug":"gpt-6.1-sol","slug":"other"`, 1), strings.Repeat(" ", 1024*1024+1)} {
		client := fixtureClient(t, "https://raw.githubusercontent.com/openai/codex/rust-v0.135.0/codex-rs/models-manager/models.json", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
		if _, err := confirmedCodex(context.Background(), "0.135.0", "gpt-6.1-sol", []string{"priority"}, client); err == nil {
			t.Fatal("unsafe catalog accepted")
		}
	}
	panicClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached network")
		return nil, nil
	})}
	for _, args := range []struct {
		version, model string
		tiers          []string
	}{{"main", "gpt-6.1-sol", nil}, {"0.135.0", "glm-5", nil}, {"0.135.0", "gpt-6-astra", []string{"turbo"}}} {
		if _, err := confirmedCodex(context.Background(), args.version, args.model, args.tiers, panicClient); err == nil {
			t.Fatal("invalid preset/version/tier accepted")
		}
	}
}

func TestInventoryFieldCaseCannotReplaceExactID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"id":"Exact","ID":"replacement"}],"DATA":[{"id":"replacement"}]}`)
	}))
	defer server.Close()
	models, err := Discover(context.Background(), server.URL, "fixture-key")
	if err != nil || len(models) != 1 || models[0].ID != "Exact" {
		t.Fatal("case-insensitive JSON field replaced exact model identity")
	}
}
func TestMetadataFieldCaseDoesNotInventCapacity(t *testing.T) {
	client := fixtureClient(t, "https://models.dev/api.json", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"provider":{"models":{"Exact":{"Limit":{"Context":999999}}}}}`)
	})
	out, err := enrich(context.Background(), []core.ModelInfo{{ID: "Exact"}}, client)
	if err != nil || len(out) != 1 || out[0].Context != 0 {
		t.Fatal("unknown schema fields invented capacity")
	}
}
func TestNativeCatalogDuplicateOtherModelRefused(t *testing.T) {
	body := strings.Replace(nativeCatalog, `{"slug":"other","service_tiers":[]}`, `{"slug":"other","service_tiers":[]},{"slug":"other","service_tiers":[]}`, 1)
	client := fixtureClient(t, "https://raw.githubusercontent.com/openai/codex/rust-v0.135.0/codex-rs/models-manager/models.json", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
	if _, err := confirmedCodex(context.Background(), "0.135.0", "gpt-6.1-sol", []string{"priority"}, client); err == nil {
		t.Fatal("duplicate native model accepted")
	}
}

func TestNetworkCancellationAndErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		code int
		kind string
	}{{401, "unauthorized"}, {403, "forbidden"}, {429, "rate_limit"}, {500, "http"}, {307, "redirect"}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
			io.WriteString(w, "fixture-private-body")
		}))
		_, err := Discover(context.Background(), server.URL, "fixture-key")
		server.Close()
		var typed *Error
		if !errors.As(err, &typed) || typed.Kind != tc.kind {
			t.Fatalf("incorrect error class for HTTP %d", tc.code)
		}
	}
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := Discover(ctx, server.URL, "fixture-key"); result <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("fixture request did not start")
	}
	cancel()
	select {
	case err := <-result:
		var typed *Error
		if !errors.As(err, &typed) || typed.Kind != "canceled" {
			t.Fatal("body cancellation not classified")
		}
	case <-time.After(time.Second):
		t.Fatal("request did not cancel")
	}
}
func TestConfirmedDefaultAndAstraAndNativeRedirect(t *testing.T) {
	url := "https://raw.githubusercontent.com/openai/codex/rust-v0.135.0/codex-rs/models-manager/models.json"
	for _, model := range []string{"gpt-6.1-sol", "gpt-6-astra"} {
		client := fixtureClient(t, url, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, strings.Replace(nativeCatalog, "gpt-6.1-sol", model, 1))
		})
		out, err := confirmedCodex(context.Background(), "0.135.0", model, []string{"default"}, client)
		if err != nil {
			t.Fatal(err)
		}
		var root struct {
			Models []map[string]json.RawMessage `json:"models"`
		}
		if json.Unmarshal(out, &root) != nil || len(root.Models) != 1 || string(root.Models[0]["service_tiers"]) != "[]" || string(root.Models[0]["additional_speed_tiers"]) != "[]" {
			t.Fatal("default tiers invented accelerated support")
		}
	}
	client := fixtureClient(t, url, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://private.invalid/fixture-secret", 302)
	})
	if _, err := confirmedCodex(context.Background(), "0.135.0", "gpt-6.1-sol", []string{"priority"}, client); err == nil || strings.Contains(err.Error(), "private.invalid") {
		t.Fatal("native redirect not refused/redacted")
	}
}
func TestDeclaredOversizeAndTransportErrorsDoNotLeak(t *testing.T) {
	for _, fixture := range []struct {
		response *http.Response
		err      error
		kind     string
	}{
		{response: &http.Response{StatusCode: 200, ContentLength: inventoryLimit + 1, Body: io.NopCloser(strings.NewReader(""))}, kind: "oversize"},
		{err: errors.New("fixture-secret https://private.invalid"), kind: "network"},
	} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return fixture.response, fixture.err })}
		_, err := discover(context.Background(), "http://localhost", "fixture-key", client)
		var typed *Error
		if !errors.As(err, &typed) || typed.Kind != fixture.kind || strings.Contains(err.Error(), "fixture-secret") || strings.Contains(err.Error(), "private.invalid") {
			t.Fatal("unsafe network error")
		}
	}
}
func TestPublicEntryPointsValidateWithoutNetwork(t *testing.T) {
	if out, err := Enrich(context.Background(), nil); err != nil || len(out) != 0 {
		t.Fatal("empty metadata needs no request")
	}
	if _, err := ConfirmedCodex(context.Background(), "main", "glm-5", nil); err == nil {
		t.Fatal("invalid public preset accepted")
	}
	if _, err := Discover(context.Background(), "http://localhost", "bad\r\nkey"); err == nil {
		t.Fatal("invalid header credential accepted")
	}
}

func TestJSONDoesNotSilentlyReplaceUnpairedSurrogateIDs(t *testing.T) {
	for _, id := range []string{`\ud800`, `\udc00`, `\ud800x`, `\ud800\u0041`} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"` + id + `"}]}`))}, nil
		})}
		if _, err := discover(context.Background(), "http://localhost", "fixture-key", client); err == nil {
			t.Fatal("malformed Unicode ID silently changed")
		}
	}
	for _, tc := range []struct{ encoded, exact string }{{`Exact-\ud83d\ude80`, "Exact-🚀"}, {`Exact-\\ud800`, `Exact-\ud800`}, {`Exact-\ufffd`, "Exact-�"}} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"` + tc.encoded + `"}]}`))}, nil
		})}
		out, err := discover(context.Background(), "http://localhost", "fixture-key", client)
		if err != nil || len(out) != 1 || out[0].ID != tc.exact {
			t.Fatal("valid Unicode ID changed")
		}
	}
}
