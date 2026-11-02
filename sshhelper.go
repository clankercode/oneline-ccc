// Package onelineccc exposes the canonical optional Windows SSH setup handoff.
package onelineccc

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
)

//go:embed scripts/setup-ssh-server.ps1
var sshScript []byte

// SSHCommand stages the canonical helper without downloading or modifying its
// body. The helper itself checks the current Windows principal's Administrator
// role before any privileged work, and retains its account/auth/firewall
// prompts. There is no automatic elevation or preapproved service change.
func SSHCommand(ctx context.Context, stateRoot string) (*exec.Cmd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		return nil, errors.New("SSH server setup is available only on Windows; use an Administrator terminal there")
	}
	if stateRoot == "" || !filepath.IsAbs(stateRoot) {
		return nil, errors.New("SSH setup requires an explicit absolute private application state root")
	}
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" || !filepath.IsAbs(systemRoot) {
		return nil, errors.New("native Windows SystemRoot is unavailable; SSH setup was not staged")
	}
	powershell := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	info, err := os.Stat(powershell)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("native Windows PowerShell is unavailable; SSH setup was not staged")
	}
	script, err := stageSSH(ctx, stateRoot)
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, powershell, "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script)
	command.Env = sshEnvironment(systemRoot)
	return command, nil
}
func stageSSH(ctx context.Context, stateRoot string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dir := filepath.Join(stateRoot, "helpers")
	if err := configtxn.PrivateDir(dir); err != nil {
		return "", err
	}
	hash := sha256.Sum256(sshScript)
	path := filepath.Join(dir, fmt.Sprintf("setup-ssh-server-%x.ps1", hash))
	before, err := configtxn.Read(path)
	if err != nil {
		return "", err
	}
	if !before.Existed {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		// Exclusive creation permits an identical concurrent winner. Any partial or
		// different existing content is refused below, never overwritten or removed.
		createErr := configtxn.PrivateFile(path, sshScript)
		before, err = configtxn.Read(path)
		if err != nil {
			return "", err
		}
		if !before.Existed {
			if createErr != nil {
				return "", createErr
			}
			return "", errors.New("SSH helper staging did not produce a protected file")
		}
	}
	if !bytes.Equal(before.Before, sshScript) || sha256.Sum256(before.Before) != hash {
		return "", errors.New("immutable SSH helper does not match the embedded canonical script; refusing execution")
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return path, nil
}
func sshEnvironment(systemRoot string) []string {
	// Never inherit ambient API keys, proxies, PowerShell module paths, shell
	// startup hooks or plugin configuration. PATH contains only native OS tools.
	env := []string{"SystemRoot=" + systemRoot, "WINDIR=" + systemRoot, "OS=Windows_NT", "COMSPEC=" + filepath.Join(systemRoot, "System32", "cmd.exe"), "PATH=" + strings.Join([]string{filepath.Join(systemRoot, "System32"), systemRoot, filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0"), filepath.Join(systemRoot, "System32", "OpenSSH")}, string(os.PathListSeparator))}
	for _, name := range []string{"USERNAME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "TEMP", "TMP", "ProgramFiles", "ProgramFiles(x86)", "ProgramData"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}
