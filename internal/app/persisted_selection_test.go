package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/core"
)

// This injected store keeps Save separate from selection, as the original
// Manager did: session > environment > vault. No native vault is accessed.
type selectionStore struct {
	session, environment, vault string
	fail                        bool
}

func (s *selectionStore) SetSession(key string) error { s.session = key; return nil }
func (s *selectionStore) Save(_ context.Context, stringKey string) error {
	if s.fail {
		return errors.New("fixture vault unavailable")
	}
	s.vault = stringKey
	return nil
}
func (s *selectionStore) Resolve(context.Context, string) (string, string, error) {
	if s.session != "" {
		status := "Session only"
		if s.session == s.vault {
			status = "Stored"
		}
		return s.session, status, nil
	}
	if s.environment != "" {
		return s.environment, "Stored", nil
	}
	return s.vault, "Stored", nil
}
func (s *selectionStore) Status(ctx context.Context, ref string) string {
	_, status, _ := s.Resolve(ctx, ref)
	return status
}
func TestFinalFixEnginePersistedSelection(t *testing.T) {
	for _, prior := range []string{"session", "environment"} {
		for _, fail := range []bool{false, true} {
			t.Run(prior+map[bool]string{false: "/save", true: "/failure"}[fail], func(t *testing.T) {
				e, _, _, _ := fixture(t, false)
				s := &selectionStore{environment: "fixture-old", fail: fail}
				if prior == "session" {
					s.session = "fixture-old"
				}
				e.deps.credentials = s
				t.Setenv("ENGINE_FIXTURE_KEY", "fixture-parent")
				err := e.SaveKey(context.Background(), "fixture-new", true)
				want := "fixture-new"
				if fail {
					want = "fixture-old"
					if err == nil {
						t.Fatal("save failure hidden")
					}
				} else if err != nil {
					t.Fatal(err)
				}
				key, status, _ := s.Resolve(context.Background(), profile().EnvKey)
				if key != want {
					t.Fatal("Engine failed to select saved key or preserve previous key")
				}
				if !fail && status != "Stored" {
					t.Fatal("persisted key labelled session only")
				}
				cmd, err := e.Command(context.Background(), "launch", "codex")
				if err != nil {
					t.Fatal(err)
				}
				selected := false
				for _, entry := range cmd.Env {
					if entry == "ENGINE_FIXTURE_KEY="+want {
						selected = true
					}
				}
				if !selected {
					t.Fatal("launch received wrong selected credential")
				}
				if strings.Contains(strings.Join(cmd.Args, " "), want) {
					t.Fatal("credential in argv")
				}
				calls := 0
				e.deps.discover = func(_ context.Context, _ string, key string) ([]core.ModelInfo, error) {
					calls++
					if key != want {
						t.Fatal("discovery received wrong credential")
					}
					return []core.ModelInfo{{ID: "fixture-model"}}, nil
				}
				e.deps.enrich = func(_ context.Context, m []core.ModelInfo) ([]core.ModelInfo, error) { return m, nil }
				if _, err := e.Discover(context.Background(), "codex"); err != nil || calls != 1 {
					t.Fatal("synthetic discovery failed")
				}
				if os.Getenv("ENGINE_FIXTURE_KEY") != "fixture-parent" {
					t.Fatal("parent environment changed")
				}
			})
		}
	}
}
