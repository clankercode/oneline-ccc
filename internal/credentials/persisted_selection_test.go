package credentials

import (
	"context"
	"errors"
	"testing"
)

func TestFinalFixPersistedSelection(t *testing.T) {
	for _, prior := range []string{"session", "environment"} {
		for _, fail := range []bool{false, true} {
			t.Run(prior+map[bool]string{false: "/save", true: "/failure"}[fail], func(t *testing.T) {
				v := &fakeVault{}
				env := map[string]string{"SELECTED_KEY": "fixture-old"}
				m := manager(v, env)
				if prior == "session" {
					if err := m.SetSession("fixture-old"); err != nil {
						t.Fatal(err)
					}
				}
				if fail {
					v.err = errors.New("fixture vault locked")
				}
				err := m.Save(context.Background(), "fixture-new")
				want := "fixture-new"
				if fail {
					want = "fixture-old"
					if err == nil {
						t.Fatal("failure hidden")
					}
				} else if err != nil {
					t.Fatal(err)
				}
				key, status, resolveErr := m.Resolve(context.Background(), "SELECTED_KEY")
				if resolveErr != nil || key != want {
					t.Fatal("explicit persisted selection or previous key lost")
				}
				if !fail && status != "Stored" {
					t.Fatal("persisted key labelled session only")
				}
				if env["SELECTED_KEY"] != "fixture-old" {
					t.Fatal("environment mutated")
				}
				if !fail {
					if err := m.SetSession("fixture-session"); err != nil {
						t.Fatal(err)
					}
					if m.Status(context.Background(), "SELECTED_KEY") != "Session only" {
						t.Fatal("new unsaved session labelled stored")
					}
				}
			})
		}
	}
}
