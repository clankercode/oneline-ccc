package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkedForeignProviderIsNotOwned(t *testing.T) {
	data := []byte(begin + "\nmodel_provider='oneline_llmp'\n[model_providers.foreign]\nname='preserve-me'\n" + end + "\n")
	if ownedProfile(data) {
		t.Fatal("foreign provider accepted as exclusively owned")
	}
	for _, data := range []string{begin + "\nmodel_provider='oneline_llmp'\nmodel_providers={oneline_llmp='wrong-type'}\n" + end,
		begin + "\nmodel_provider='oneline_llmp'\nfeatures='wrong-type'\n[model_providers.oneline_llmp]\nname='test'\n" + end} {
		if ownedProfile([]byte(data)) {
			t.Fatal("malformed nested shape accepted")
		}
	}
}
func TestNativeImportedOverridesAreNotAttestation(t *testing.T) {
	a := nativeAdapter(t, false)
	data := "model='gpt-6.1-sol'\nmodel_provider='oneline_llmp'\nmodel_context_window=500000\nmodel_auto_compact_token_limit=450000\nservice_tier='ultrafast'\nfeatures.ultrafast_mode=true\n[model_providers.oneline_llmp]\nname='fixture'\nbase_url='http://127.0.0.1:9/v1'\nenv_key='LLMP_API_KEY'\nwire_api='responses'\n"
	if err := os.WriteFile(filepath.Join(a.Home, "config.toml"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := a.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c.Profile.CapacityConfirmed || c.Profile.TiersConfirmed {
		t.Fatal("native overrides fabricated operator confirmation")
	}
}
func TestNativeStandardCatalogRequiresFullSchema(t *testing.T) {
	a := nativeAdapter(t, false)
	p := fixtureProfile()
	bad := []byte(`{"models":[{"slug":"gpt-6.1-sol"}]}`)
	if _, err := a.Prepare(context.Background(), p, bad); err == nil {
		t.Fatal("incomplete Standard catalog accepted")
	}
	path := filepath.Join(a.Home, "incomplete.json")
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	p.Catalog = path
	if _, err := a.Prepare(context.Background(), p, nil); err == nil {
		t.Fatal("incomplete external Standard catalog accepted")
	}
	if full := os.Getenv("ONELINE_CODEX_TEST_CATALOG"); full != "" {
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		p.Catalog = ""
		if _, err = a.Prepare(context.Background(), p, data); err != nil {
			t.Fatal("complete Standard catalog refused", err)
		}
	}
}
