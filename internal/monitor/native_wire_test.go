package monitor

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFinalFixNativeBusctlArrays(t *testing.T) {
	for _, kind := range []string{"multiple paths", "single path", "command"} {
		t.Run(kind, func(t *testing.T) {
			m, f := setupService(t)
			signature := "as"
			var data any = f.unitPaths
			if kind == "single path" {
				data = f.unitPaths[:1]
			}
			if kind == "command" {
				signature = "a(sasbttttuii)"
				data = []any{[]any{m.executable, m.argv(), false, 0, 0, 0, 0, 0, 0, 0}}
			}
			raw, _ := json.Marshal(map[string]any{"type": signature, "data": data})
			m.runner = func(context.Context, string, ...string) (commandResult, error) {
				return commandResult{Output: string(raw)}, nil
			}
			var err error
			if kind == "command" {
				err = m.effectiveCommand(context.Background())
			} else {
				spec, _ := m.spec()
				_, err = m.systemdPaths(context.Background(), spec)
			}
			if err != nil {
				t.Fatalf("native direct data array rejected: %v", err)
			}
		})
	}
}

func TestFinalFixMalformedBusctlArrays(t *testing.T) {
	for _, property := range []string{"UnitPath", "ExecStart"} {
		for _, raw := range []string{`{}`, `{"type":"s","data":["wrong"]}`, `{"type":"as","data":null}`, `{"type":"as","data":"path"}`, `{"type":"as","data":[["path"]]}`, `{"type":"as","data":[1]}`, `{"type":"as","data":[]}`, `{"type":"a(sasbttttuii)","data":[]}`, `{"type":"a(sasbttttuii)","data":[["path",[],false]]}`} {
			t.Run(property+raw, func(t *testing.T) {
				m, _ := setupService(t)
				m.runner = func(context.Context, string, ...string) (commandResult, error) {
					return commandResult{Output: raw}, nil
				}
				var err error
				if property == "ExecStart" {
					err = m.effectiveCommand(context.Background())
				} else {
					spec, _ := m.spec()
					_, err = m.systemdPaths(context.Background(), spec)
				}
				if err == nil {
					t.Fatal("malformed/wrong signature accepted")
				}
			})
		}
	}
}

func TestFinalFixExecStartTypedFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field int
		value any
	}{
		{"null ignore flag", 2, nil}, {"string ignore flag", 2, "false"}, {"ignore failures", 2, true},
		{"negative timestamp", 3, -1}, {"null timestamp", 4, nil}, {"string timestamp", 5, "0"},
		{"timestamp overflow", 6, json.RawMessage("18446744073709551616")},
		{"pid overflow", 7, json.RawMessage("4294967296")}, {"null pid", 7, nil},
		{"code overflow", 8, json.RawMessage("2147483648")}, {"status overflow", 9, json.RawMessage("-2147483649")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := setupService(t)
			entry := []any{m.executable, m.argv(), false, 0, 0, 0, 0, 0, 0, 0}
			entry[tc.field] = tc.value
			raw, err := json.Marshal(map[string]any{"type": "a(sasbttttuii)", "data": []any{entry}})
			if err != nil {
				t.Fatal(err)
			}
			m.runner = func(context.Context, string, ...string) (commandResult, error) {
				return commandResult{Output: string(raw)}, nil
			}
			if err := m.effectiveCommand(context.Background()); err == nil {
				t.Fatal("malformed native tuple field accepted")
			}
		})
	}
}
