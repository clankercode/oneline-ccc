package monitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewPermissionRevocationAtQueryBoundary(t *testing.T) {
	for _, operation := range []string{"status", "start"} {
		t.Run(operation, func(t *testing.T) {
			m, f := setupService(t)
			if _, err := m.Action(context.Background(), "install"); err != nil {
				t.Fatal(err)
			}
			queries, before := 0, mutationCount(f)
			m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
				result, err := f.run(ctx, name, args...)
				if len(args) > 3 && args[3] == "show" {
					queries++
					boundary := 2
					if operation == "status" {
						boundary = 1
					}
					if queries == boundary {
						if err := os.Chmod(m.stateRoot, 0777); err != nil {
							t.Fatal(err)
						}
					}
				}
				return result, err
			}
			if operation == "status" {
				info := m.Status(context.Background())
				if info.Status != "Conflict" {
					t.Fatalf("revoked directory: published %s", info.Status)
				}
			} else {
				info, err := m.Action(context.Background(), "start")
				if mutationCount(f) != before || f.active {
					t.Fatalf("revoked directory: mutations=%d active=%t result=%s err=%v", mutationCount(f)-before, f.active, info.Status, err)
				}
			}
		})
	}
}
func TestReviewUninstallMissingDefinitionRetainsActiveRegistration(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		t.Run(fmt.Sprintf("pending_%t", reserved), func(t *testing.T) {
			m, f := setupService(t)
			if _, err := m.Action(context.Background(), "install"); err != nil {
				t.Fatal(err)
			}
			if _, err := m.Action(context.Background(), "start"); err != nil {
				t.Fatal(err)
			}
			spec, _ := m.spec()
			if err := os.Remove(spec.artifact); err != nil {
				t.Fatal(err)
			}
			f.loaded = false
			f.fragment = ""
			if reserved {
				if err := atomicRecord(context.Background(), filepath.Join(m.root(), "pending.json"), pendingAction{Version: 1, Action: "uninstall"}); err != nil {
					t.Fatal(err)
				}
			}
			before := mutationCount(f)
			info, err := m.Action(context.Background(), "uninstall")
			_, pendingErr := os.Stat(filepath.Join(m.root(), "pending.json"))
			if err == nil && (f.active || f.enabled) {
				t.Fatalf("successful %s with active=%t enabled=%t mutations=%d pending_exists=%t", info.Status, f.active, f.enabled, mutationCount(f)-before, pendingErr == nil)
			}
		})
	}
}
func TestReviewUninstallRuntimeEnabledRegistration(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
		result, err := f.run(ctx, name, args...)
		result.Output = strings.Replace(result.Output, "UnitFileState=enabled\n", "UnitFileState=enabled-runtime\n", 1)
		return result, err
	}
	before := mutationCount(f)
	info, err := m.Action(context.Background(), "uninstall")
	if err == nil && f.enabled {
		t.Fatalf("runtime registration retained: result=%s mutations=%d enabled=%t", info.Status, mutationCount(f)-before, f.enabled)
	}
}
func TestReviewForeignSystemdDropInPreventsStart(t *testing.T) {
	m, f := setupService(t)
	spec, _ := m.spec()
	dropInDir := spec.artifact + ".d"
	if err := os.MkdirAll(dropInDir, 0700); err != nil {
		t.Fatal(err)
	}
	foreign := []byte("[Service]\nExecStart=\nExecStart=/synthetic/foreign-program\n")
	if err := os.WriteFile(filepath.Join(dropInDir, "override.conf"), foreign, 0600); err != nil {
		t.Fatal(err)
	}
	installInfo, installErr := m.Action(context.Background(), "install")
	if installErr != nil {
		return
	}
	before := mutationCount(f)
	info, err := m.Action(context.Background(), "start")
	if err == nil || mutationCount(f) != before {
		t.Fatalf("foreign drop-in accepted: install=%s start=%s err=%v start_mutations=%d", installInfo.Status, info.Status, err, mutationCount(f)-before)
	}
}

func TestReviewF1AllNativePathsPrefixesAndCachedDefinition(t *testing.T) {
	for _, location := range []string{"user", "runtime", "vendor", "control", "prefix", "type-wide", "cached-dropin", "cached-command"} {
		t.Run(location, func(t *testing.T) {
			m, f := setupService(t)
			var foreignPath string
			foreign := []byte("[Service]\nExecStart=\nExecStart=/synthetic/foreign-program\n")
			if strings.HasPrefix(location, "cached-") {
				if _, err := m.Action(context.Background(), "install"); err != nil {
					t.Fatal(err)
				}
				if location == "cached-dropin" {
					f.dropInPaths = "/synthetic/cached-foreign.conf"
				} else {
					f.effectiveArgv = []string{"/synthetic/foreign-program"}
				}
			} else {
				index := map[string]int{"user": 0, "runtime": 1, "vendor": 2, "control": 3}[location]
				name := unitName + ".d"
				if location == "prefix" {
					name = "oneline-ccc-.service.d"
				}
				if location == "type-wide" {
					name = "service.d"
				}
				foreignPath = filepath.Join(f.unitPaths[index], name, "override.conf")
				if err := os.MkdirAll(filepath.Dir(foreignPath), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(foreignPath, foreign, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := mutationCount(f)
			action := "install"
			if strings.HasPrefix(location, "cached-") {
				action = "start"
			}
			info, err := m.Action(context.Background(), action)
			if err == nil || info.Status != "Conflict" || mutationCount(f) != before {
				t.Fatalf("%s foreign effective definition accepted: %s %v", location, info.Status, err)
			}
			if foreignPath != "" {
				after, _ := os.ReadFile(foreignPath)
				if string(after) != string(foreign) {
					t.Fatal("foreign drop-in changed")
				}
			}
		})
	}
}

func TestReviewF2EachMissingDefinitionEvidenceRetainsPending(t *testing.T) {
	for _, evidence := range []string{"active", "enabled", "invocation", "transition"} {
		t.Run(evidence, func(t *testing.T) {
			m, f := setupService(t)
			if _, err := m.Action(context.Background(), "install"); err != nil {
				t.Fatal(err)
			}
			spec, _ := m.spec()
			if err := os.Remove(spec.artifact); err != nil {
				t.Fatal(err)
			}
			f.loaded = false
			f.fragment = ""
			f.active = false
			f.enabled = false
			f.invocation = ""
			switch evidence {
			case "active":
				f.active = true
				f.invocation = strings.Repeat("1", 32)
			case "enabled":
				f.enabled = true
			case "invocation":
				f.invocation = strings.Repeat("2", 32)
			}
			if evidence == "transition" {
				m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
					result, err := f.run(ctx, name, args...)
					result.Output = strings.Replace(result.Output, "ActiveState=inactive", "ActiveState=deactivating", 1)
					return result, err
				}
			}
			path := filepath.Join(m.root(), "pending.json")
			if err := atomicRecord(context.Background(), path, pendingAction{Version: 1, Action: "uninstall"}); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(path)
			mutations := mutationCount(f)
			state, err := m.query(context.Background(), spec)
			if err != nil || state.loaded || !state.registrationEvidence() {
				t.Fatal("not-found erased registration evidence")
			}
			info, err := m.Action(context.Background(), "uninstall")
			after, _ := os.ReadFile(path)
			if err == nil || info.Status != "Conflict" && info.Status != "Unknown" || string(after) != string(before) || mutationCount(f) != mutations {
				t.Fatal("unverified missing-definition uninstall changed reservation/native state")
			}
		})
	}
}
func TestReviewF3RuntimeEnablementConflictPreservesAllState(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	spec, _ := m.spec()
	before, _ := os.ReadFile(spec.artifact)
	m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
		result, err := f.run(ctx, name, args...)
		result.Output = strings.Replace(result.Output, "UnitFileState=enabled\n", "UnitFileState=enabled-runtime\n", 1)
		return result, err
	}
	mutations := mutationCount(f)
	for _, action := range []string{"install", "start", "stop", "restart", "uninstall"} {
		info, err := m.Action(context.Background(), action)
		if err == nil || info.Status != "Conflict" || !strings.Contains(info.Detail, "runtime") {
			t.Fatal("runtime enablement not explicitly refused")
		}
	}
	info := m.Status(context.Background())
	after, _ := os.ReadFile(spec.artifact)
	if info.Status != "Conflict" || !f.enabled || string(before) != string(after) || mutationCount(f) != mutations {
		t.Fatal("runtime enablement or owned definition changed")
	}
}
