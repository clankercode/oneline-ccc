package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const systemdBus = "org.freedesktop.systemd1"
const systemdManagerPath = "/org/freedesktop/systemd1"
const systemdUnitPath = "/org/freedesktop/systemd1/unit/oneline_2dccc_2dobserver_2eservice"

// busctl's typed JSON preserves paths/argv containing spaces without shell or
// presentation-string parsing. Missing/unrecognized native evidence fails closed.
func (m *Manager) busProperty(ctx context.Context, path, iface, property, signature string) (json.RawMessage, error) {
	result, err := m.nativeCall(ctx, "/usr/bin/busctl", "--user", "--json=short", "get-property", systemdBus, path, iface, property)
	if err == errUnsafe {
		return nil, err
	}
	if err != nil || result.Code != 0 || len(result.Output) > recordLimit || !strictJSON([]byte(result.Output)) {
		return nil, errUnavailable
	}
	var response struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal([]byte(result.Output), &response) != nil || response.Type != signature || len(response.Data) == 0 || response.Data[0] != '[' {
		return nil, errUnavailable
	}
	return response.Data, nil
}
func (m *Manager) systemdPaths(ctx context.Context, spec serviceSpec) ([]string, error) {
	raw, err := m.busProperty(ctx, systemdManagerPath, systemdBus+".Manager", "UnitPath", "as")
	if err != nil {
		return nil, err
	}
	var paths []string
	if json.Unmarshal(raw, &paths) != nil || len(paths) == 0 || len(paths) > 128 {
		return nil, errUnavailable
	}
	found := false
	seen := map[string]bool{}
	unique := []string{}
	for _, path := range paths {
		if !safePath(path) {
			return nil, errUnsafe
		}
		if path == filepath.Dir(spec.artifact) {
			found = true
		}
		if !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}
	if !found {
		return nil, errUnsafe
	}
	if err = m.checkSystemdPaths(spec, unique); err != nil {
		return nil, err
	}
	return unique, nil
}
func (m *Manager) effectiveCommand(ctx context.Context) error {
	raw, err := m.busProperty(ctx, systemdUnitPath, systemdBus+".Service", "ExecStart", "a(sasbttttuii)")
	if err != nil {
		return err
	}
	var entries []json.RawMessage
	var entry []json.RawMessage
	if json.Unmarshal(raw, &entries) != nil || len(entries) != 1 || json.Unmarshal(entries[0], &entry) != nil || len(entry) != 10 {
		return errForeignOverride
	}
	var path string
	var argv []string
	if json.Unmarshal(entry[0], &path) != nil || json.Unmarshal(entry[1], &argv) != nil || !bytes.Equal(bytes.TrimSpace(entry[2]), []byte("false")) || path != m.executable || !slices.Equal(argv, m.argv()) {
		return errForeignOverride
	}
	// The remaining fields are four timestamps, pid, code and status. Validate
	// their native types too; a matching argv cannot bless a malformed struct.
	for _, field := range entry[3:7] {
		var stamp uint64
		if bytes.Equal(bytes.TrimSpace(field), []byte("null")) || json.Unmarshal(field, &stamp) != nil {
			return errForeignOverride
		}
	}
	var pid uint32
	if bytes.Equal(bytes.TrimSpace(entry[7]), []byte("null")) || json.Unmarshal(entry[7], &pid) != nil {
		return errForeignOverride
	}
	for _, field := range entry[8:] {
		var status int32
		if bytes.Equal(bytes.TrimSpace(field), []byte("null")) || json.Unmarshal(field, &status) != nil {
			return errForeignOverride
		}
	}
	return nil
}

// systemd applies type-wide and dash-prefix drop-ins in every manager UnitPath,
// including user/runtime/control/generator/vendor paths. A not-yet-reloaded file
// is absent from DropInPaths, so native properties alone are insufficient.
func dropInNames() []string {
	names := []string{unitName + ".d", "service.d"}
	stem := strings.TrimSuffix(unitName, ".service")
	for {
		index := strings.LastIndex(stem, "-")
		if index < 0 {
			break
		}
		names = append(names, stem[:index+1]+".service.d")
		stem = stem[:index]
	}
	return names
}
func (m *Manager) checkSystemdPaths(spec serviceSpec, paths []string) error {
	for _, root := range paths {
		main := filepath.Join(root, unitName)
		if safeComponents(main) != nil {
			return errForeignOverride
		}
		if _, err := os.Lstat(main); err == nil {
			if main != spec.artifact {
				return errForeignOverride
			}
		} else if !os.IsNotExist(err) {
			return errUnsafe
		}
		for _, name := range dropInNames() {
			if err := noDropIns(filepath.Join(root, name)); err != nil {
				return err
			}
		}
	}
	return nil
}
func noDropIns(path string) error {
	if safeComponents(path) != nil {
		return errForeignOverride
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.IsDir() {
		return errForeignOverride
	}
	f, err := os.OpenFile(path, os.O_RDONLY|lockFlags(), 0)
	if err != nil {
		return errForeignOverride
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errForeignOverride
	}
	for count := 0; count < 256; count++ {
		entries, err := f.ReadDir(1)
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".conf") {
				return errForeignOverride
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errForeignOverride
		}
	}
	return errForeignOverride
}
