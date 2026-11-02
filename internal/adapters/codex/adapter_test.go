package codex

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func nativeAdapter(t *testing.T, fresh bool) *Adapter {
	t.Helper()
	binary := os.Getenv("ONELINE_CODEX_TEST_BINARY")
	if binary == "" {
		t.Skip("requires an explicit isolated native Codex fixture")
	}
	root := t.TempDir()
	home := filepath.Join(root, "codex")
	if !fresh {
		if err := os.Mkdir(home, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return New(home, filepath.Join(root, "scratch"), binary)
}
func fixtureProfile() core.Profile {
	p := core.Defaults()
	p.BaseURL = "http://127.0.0.1:9/v1"
	p.CapacityConfirmed = true
	return p
}
func TestNativeFreshPrepareDoesNotCreateClientHome(t *testing.T) {
	a := nativeAdapter(t, true)
	c, err := a.Inspect(context.Background())
	if err != nil || !c.Supported {
		t.Fatalf("inspect: %+v %v", c, err)
	}
	writes, err := a.Prepare(context.Background(), fixtureProfile(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(writes) != 2 {
		t.Fatalf("want root and named profile, got%d", len(writes))
	}
	if _, err = os.Stat(a.Home); !os.IsNotExist(err) {
		t.Fatal("planning created client home")
	}
	for _, w := range writes {
		if w.Existed || len(w.Before) != 0 || len(w.Data) == 0 {
			t.Fatal("bad fresh snapshot")
		}
	}
}
func TestNativePreservationApplyAndNoop(t *testing.T) {
	a := nativeAdapter(t, false)
	before := []byte("# foreign comment must remain\nmodel = \"fixture-old\"\n\n[history] # history table\npersistence = \"none\" # preserve me\n\n[notice]\nhide_rate_limit_model_nudge = true\n")
	path := filepath.Join(a.Home, "config.toml")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	writes, err := a.Prepare(context.Background(), fixtureProfile(), nil)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(path)
	if !bytes.Equal(before, current) {
		t.Fatal("plan changed real config")
	}
	for _, s := range []string{"# foreign comment must remain", "[history] # history table", "# preserve me", "hide_rate_limit_model_nudge = true"} {
		if !bytes.Contains(writes[0].Data, []byte(s)) {
			t.Errorf("lost %q", s)
		}
	}
	store := configtxn.New(filepath.Join(filepath.Dir(a.Home), "transactions"))
	result, err := store.Apply(context.Background(), writes)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "Succeeded" {
		t.Fatalf("outcome %s", result.Outcome)
	}
	c, err := a.Inspect(context.Background())
	if err != nil || c.Profile.Model != fixtureProfile().Model || c.Profile.Tier != "Standard" || c.Profile.Budget != 500000 {
		t.Fatalf("effective %+v %v", c, err)
	}
	confirmed := c.Profile
	confirmed.CapacityConfirmed = true // Adapter imports alone are not route evidence.
	again, err := a.Prepare(context.Background(), confirmed, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range again {
		if !bytes.Equal(w.Before, w.Data) {
			t.Fatalf("rerun changed %s", filepath.Base(w.Path))
		}
	}
}
func TestNativeForeignProfileAndInlineCredentialsRefused(t *testing.T) {
	for _, kind := range []string{"foreign-profile", "inline-secret", "conflicting-auth"} {
		t.Run(kind, func(t *testing.T) {
			a := nativeAdapter(t, false)
			path := filepath.Join(a.Home, "config.toml")
			data := []byte("# preserve\n")
			if kind == "foreign-profile" {
				if err := os.WriteFile(filepath.Join(a.Home, "llmp.config.toml"), []byte("# operator\nmodel='custom'\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "inline-secret" {
				data = []byte("[model_providers.other]\nexperimental_bearer_token='fixture-private-marker'\n")
			}
			if kind == "conflicting-auth" {
				data = []byte("[model_providers.oneline_llmp]\nenv_http_headers = { X-API-Key = 'OTHER_KEY' }\n")
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			_, err := a.Prepare(context.Background(), fixtureProfile(), nil)
			if err == nil {
				t.Fatal("unsafe adoption accepted")
			}
			if strings.Contains(err.Error(), "fixture-private-marker") {
				t.Fatal("error leaked inline credential")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(data, after) {
				t.Fatal("refusal modified input")
			}
		})
	}
}

func TestNativeFullTierCatalog(t *testing.T) {
	path := os.Getenv("ONELINE_CODEX_TEST_CATALOG")
	if path == "" {
		t.Skip("requires explicit version-matched public catalog fixture")
	}
	a := nativeAdapter(t, false)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p := fixtureProfile()
	p.TiersConfirmed = true
	p.Tier = "Ultrafast"
	writes, err := a.Prepare(context.Background(), p, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(writes) != 2 || !bytes.Contains(writes[0].Data, []byte("ultrafast")) {
		t.Fatal("missing tier configuration")
	}
	// A skeletal picker document must not pass the native metadata check.
	_, err = a.Prepare(context.Background(), p, []byte(`{"models":[{"slug":"gpt-6.1-sol","visibility":"list","supported_in_api":true,"service_tiers":[]}]}`))
	if err == nil {
		t.Fatal("incomplete native model schema accepted")
	}
}
