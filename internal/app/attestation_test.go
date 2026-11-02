package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestImportedNativeFlagsAreNotAttestation(t *testing.T) {
	e, a, _, o := fixture(t, false)
	a.profile.CapacityConfirmed = true
	a.profile.TiersConfirmed = true
	s, err := e.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Clients[0].Profile.CapacityConfirmed || s.Clients[0].Profile.TiersConfirmed {
		t.Fatal("native overrides were treated as operator attestation")
	}
	desired := s.Clients[0].Profile
	if _, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: desired}); err == nil {
		t.Fatal("import enabled without fresh capacity confirmation")
	}
	// Legacy managed state contains no proof, even when native overrides exist.
	if err := configtxn.PrivateFile(filepath.Join(o.StateRoot, "settings.json"), []byte(`{"version":1,"managed":{"codex":true},"receipts":[]}`)); err != nil {
		t.Fatal(err)
	}
	s, err = e.Inspect(context.Background())
	if err != nil || s.Clients[0].Profile.CapacityConfirmed || s.Clients[0].Profile.TiersConfirmed {
		t.Fatal("legacy management fabricated attestation")
	}
}
func TestAttestedApplyRestoresOnlyExactManagedFingerprintAndNoop(t *testing.T) {
	e, a, _, o := fixture(t, false)
	p := profile()
	p.Model = "gpt-6.1-sol"
	p.Tier = "Ultrafast"
	p.TiersConfirmed = true
	a.profile = p
	e.deps.confirmed = func(context.Context, string, string, []string) ([]byte, error) { return nil, nil }
	reviewed := plan(t, e, core.Desired{Enabled: true, Profile: p})
	beforeApply, err := e.Inspect(context.Background())
	if err != nil || beforeApply.Clients[0].Profile.CapacityConfirmed || beforeApply.Clients[0].Profile.TiersConfirmed {
		t.Fatal("Plan persisted attestation before Apply")
	}
	if _, err := e.Apply(context.Background(), reviewed.ID, reviewed.Revision); err != nil {
		t.Fatal(err)
	}
	a.profile.CapacityConfirmed = false
	a.profile.TiersConfirmed = false
	// Proof must survive a fresh engine, rather than depend on pending memory.
	e, err = newEngine(o, e.deps)
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := s.Clients[0].Profile
	if !got.CapacityConfirmed || !got.TiersConfirmed {
		t.Fatal("successful reviewed Apply did not persist hashed confirmation")
	}
	path := filepath.Join(o.StateRoot, "settings.json")
	before, _ := os.ReadFile(path)
	var stored map[string]json.RawMessage
	if json.Unmarshal(before, &stored) != nil || len(stored["attestations"]) == 0 || bytes.Contains(before, []byte(p.BaseURL)) || bytes.Contains(before, []byte(p.Model)) {
		t.Fatal("settings lack private hashed evidence or persist raw profile")
	}
	// Even a caller holding the native adapter's unconfirmed view must retain
	// existing proven confirmations for exactly the same values.
	got.CapacityConfirmed = false
	got.TiersConfirmed = false
	requested := core.Desired{Enabled: true, Profile: got}
	same := plan(t, e, requested)
	if !same.Noop || same.Desired != requested {
		t.Fatal("exact no-op lost proof or changed the frozen request")
	}
	if _, err := e.Apply(context.Background(), same.ID, same.Revision); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("attested no-op rewrote evidence or receipts")
	}
	// A changed catalog selected during staging cannot inherit old proof.
	a.prepare = func(context.Context, core.Profile, []byte) ([]configtxn.Write, error) {
		w, err := configtxn.Read(filepath.Join(a.home, "config.toml"))
		w.Data = []byte("model_catalog_json = \"/fixture/replacement.json\"\n")
		return []configtxn.Write{w}, err
	}
	if _, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: got}); err == nil {
		t.Fatal("staged catalog drift inherited old confirmation")
	}
	a.prepare = nil
	for _, change := range []string{"endpoint", "model", "budget", "compact", "tier", "env", "catalog", "version"} {
		t.Run(change, func(t *testing.T) {
			old := a.profile
			a.profile = old
			switch change {
			case "endpoint":
				a.profile.BaseURL = "https://different.invalid/v1"
			case "model":
				a.profile.Model = "other-exact-model"
			case "budget":
				a.profile.Budget++
			case "compact":
				a.profile.Compact++
			case "tier":
				a.profile.Tier = "Fast"
			case "env":
				a.profile.EnvKey = "OTHER_FIXTURE_KEY"
			case "catalog":
				a.profile.Catalog = "/different/catalog.json"
			case "version":
				a.inspect = func(context.Context) (core.Client, error) {
					return core.Client{Installed: true, Supported: true, Version: "0.167.0", Profile: a.profile}, nil
				}
			}
			state, err := e.Inspect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if state.Clients[0].Profile.CapacityConfirmed || state.Clients[0].Profile.TiersConfirmed {
				t.Fatal("changed native profile retained confirmation")
			}
			a.profile = old
			a.inspect = nil
		})
	}
	off := plan(t, e, core.Desired{Enabled: false})
	if _, err := e.Apply(context.Background(), off.ID, off.Revision); err != nil {
		t.Fatal(err)
	}
	s, err = e.Inspect(context.Background())
	if err != nil || s.Clients[0].Profile.CapacityConfirmed || s.Clients[0].Profile.TiersConfirmed {
		t.Fatal("unmanaged client retained confirmations")
	}
}

func TestCapacityEvidenceDoesNotInventTierConfirmation(t *testing.T) {
	e, a, _, _ := fixture(t, false)
	p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	if _, err := e.Apply(context.Background(), p.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	a.profile.CapacityConfirmed = false
	a.profile.TiersConfirmed = true // Native flags have no authority.
	s, err := e.Inspect(context.Background())
	if err != nil || !s.Clients[0].Profile.CapacityConfirmed || s.Clients[0].Profile.TiersConfirmed {
		t.Fatal("capacity-only proof restored unconfirmed tiers")
	}
}
func TestAttestationBindsActualStagedCatalogPath(t *testing.T) {
	e, a, _, _ := fixture(t, false)
	p := profile()
	p.Model = "gpt-6.1-sol"
	p.TiersConfirmed = true
	e.deps.confirmed = func(context.Context, string, string, []string) ([]byte, error) {
		return []byte("fixture-complete-catalog"), nil
	}
	a.prepare = func(context.Context, core.Profile, []byte) ([]configtxn.Write, error) {
		return []configtxn.Write{{Path: filepath.Join(a.home, "config.toml"), Mode: 0600, Data: []byte("model_catalog_json = \"/fixture/validated-catalog.json\"\n")}}, nil
	}
	reviewed := plan(t, e, core.Desired{Enabled: true, Profile: p})
	if _, err := e.Apply(context.Background(), reviewed.ID, reviewed.Revision); err != nil {
		t.Fatal(err)
	}
	a.profile = p
	a.profile.Catalog = "/fixture/validated-catalog.json"
	a.profile.CapacityConfirmed = false
	a.profile.TiersConfirmed = false
	s, err := e.Inspect(context.Background())
	if err != nil || !s.Clients[0].Profile.CapacityConfirmed || !s.Clients[0].Profile.TiersConfirmed {
		t.Fatal("attestation bound caller catalog hint instead of native staged path")
	}
}
