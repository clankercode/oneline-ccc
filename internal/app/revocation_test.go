package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type waitingTransaction struct {
	*configtxn.Store
	entered chan context.Context
	release chan struct{}
}

func (s waitingTransaction) Apply(ctx context.Context, _ []configtxn.Write) (configtxn.Result, error) {
	s.entered <- ctx
	<-s.release
	if err := ctx.Err(); err != nil {
		return configtxn.Result{Outcome: "Cancelled"}, err
	}
	return configtxn.Result{Outcome: "Succeeded"}, nil
}

func TestSupersedingPlanCancelsContextAlreadyInsideTransaction(t *testing.T) {
	e, _, _, _ := fixture(t, false)
	old := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	store := waitingTransaction{entered: make(chan context.Context, 1), release: make(chan struct{})}
	e.store = store
	done := make(chan error, 1)
	go func() { _, err := e.Apply(context.Background(), old.ID, old.Revision); done <- err }()
	var operation context.Context
	select {
	case operation = <-store.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("Apply never reached transaction")
	}
	p := profile()
	p.Model = "newer-reviewed-model"
	plan(t, e, core.Desired{Enabled: true, Profile: p})
	if operation.Err() != context.Canceled {
		t.Error("new Plan did not revoke the context already passed to Store.Apply")
	}
	close(store.release)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("superseded transaction succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("revoked Apply did not finish")
	}
}

func TestDiscoveryRechecksProtectedStateBeforeEveryProjection(t *testing.T) {
	for _, branch := range []string{"enriched", "metadata-error", "different-id"} {
		t.Run(branch, func(t *testing.T) {
			e, _, _, o := fixture(t, false)
			e.deps.discover = func(context.Context, string, string) ([]core.ModelInfo, error) {
				return []core.ModelInfo{{ID: "exact-fixture-model"}}, nil
			}
			e.deps.enrich = func(_ context.Context, models []core.ModelInfo) ([]core.ModelInfo, error) {
				if err := configtxn.PrivateFile(filepath.Join(o.StateRoot, "settings.json"), []byte("invalid fixture state")); err != nil {
					t.Fatal(err)
				}
				switch branch {
				case "metadata-error":
					return nil, errors.New("fixture unavailable")
				case "different-id":
					return []core.ModelInfo{{ID: "different-fixture-model"}}, nil
				}
				return models, nil
			}
			models, err := e.Discover(context.Background(), "codex")
			if err == nil || len(models) != 0 {
				t.Fatal("inventory projected after protected state became invalid")
			}
		})
	}
}

func TestPlanPreservesExactRequestedDesiredWithGeneratedCatalog(t *testing.T) {
	e, a, _, _ := fixture(t, false)
	p := profile()
	p.Model = "gpt-6.1-sol"
	p.Tier = "ultrafast"
	p.TiersConfirmed = true
	want := core.Desired{Enabled: true, Profile: p}
	e.deps.confirmed = func(context.Context, string, string, []string) ([]byte, error) {
		return []byte("fixture-complete-catalog"), nil
	}
	a.prepare = func(context.Context, core.Profile, []byte) ([]configtxn.Write, error) {
		return []configtxn.Write{{Path: filepath.Join(a.home, "config.toml"), Mode: 0600, Data: []byte("model_catalog_json = \"/fixture/generated-catalog.json\"\n")}}, nil
	}
	pending := plan(t, e, want)
	if pending.Desired != want {
		t.Fatal("public Plan.Desired changed the exact frozen request")
	}
}
