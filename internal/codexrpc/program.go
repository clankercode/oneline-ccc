package codexrpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FindProgram bypasses token-loading wrappers; only native executable headers
// are accepted. An explicit path remains subject to the same checks.
func FindProgram(explicit string) (Program, error) {
	if explicit != "" {
		return nativeProgram(explicit)
	}
	path, _ := exec.LookPath("codex")
	if p, err := nativeProgram(path); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	platform := map[string]string{"linux": "linux", "windows": "win32", "darwin": "darwin"}[runtime.GOOS]
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	roots := []string{filepath.Join(filepath.Dir(path), "node_modules", "@openai"), filepath.Join(home, ".bun", "install", "global", "node_modules", "@openai"), filepath.Join(home, ".npm-global", "lib", "node_modules", "@openai"), "/usr/local/lib/node_modules/@openai", "/usr/lib/node_modules/@openai"}
	for _, root := range roots {
		for _, base := range []string{filepath.Join(root, "codex-"+platform+"-"+arch, "vendor"), filepath.Join(root, "codex", "node_modules", "@openai", "codex-"+platform+"-"+arch, "vendor"), filepath.Join(root, "codex", "vendor")} {
			for _, layout := range []string{"bin", "codex"} {
				matches, _ := filepath.Glob(filepath.Join(base, "*", layout, name))
				for _, match := range matches {
					if p, err := nativeProgram(match); err == nil {
						return p, nil
					}
				}
			}
		}
	}
	return Program{}, errors.New("native Codex executable not found; install Codex or select its native executable path")
}
func nativeProgram(path string) (Program, error) {
	if path == "" {
		return Program{}, errors.New("missing Codex executable")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Program{}, errors.New("Codex executable unavailable")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return Program{}, err
	}
	f, err := os.Open(resolved)
	if err != nil {
		return Program{}, errors.New("Codex executable unreadable")
	}
	defer f.Close()
	var header [4]byte
	if _, err = io.ReadFull(f, header[:]); err != nil {
		return Program{}, errors.New("Codex executable is not a native binary")
	}
	valid := string(header[:]) == "\x7fELF" || string(header[:2]) == "MZ" || header == [4]byte{0xcf, 0xfa, 0xed, 0xfe} || header == [4]byte{0xca, 0xfe, 0xba, 0xbe} || header == [4]byte{0xfe, 0xed, 0xfa, 0xcf}
	if !valid {
		return Program{}, errors.New("Codex wrapper refused; choose the underlying native binary")
	}
	return Program{Path: resolved}, nil
}
func Version(ctx context.Context, p Program, stage string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, p.Path, append(append([]string{}, p.Args...), "--version")...)
	c.Env = ConfigEnvironment(stage)
	c.Dir = stage
	data, err := c.Output()
	if err != nil {
		return "", errors.New("Codex version probe failed")
	}
	value := strings.TrimSpace(string(data))
	if !strings.HasPrefix(value, "codex-cli ") {
		return "", errors.New("unsupported Codex identity")
	}
	return strings.TrimPrefix(value, "codex-cli "), nil
}
func Identity(p Program) (string, error) {
	f, err := os.Open(p.Path)
	if err != nil {
		return "", errors.New("selected Codex executable is unavailable")
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, 512<<20))
	if err != nil || n == 512<<20 {
		return "", errors.New("cannot verify selected Codex executable")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
