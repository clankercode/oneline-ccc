package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
)

func TestStageReplacementProcess(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "-- replace-stage") {
		return
	}
	stage, err := os.Getwd()
	if err != nil || os.Chdir(filepath.Dir(stage)) != nil || os.Rename(stage, stage+"-original") != nil {
		os.Exit(41)
	}
	if configtxn.PrivateDir(stage) != nil || configtxn.PrivateFile(filepath.Join(stage, "foreign"), []byte("preserve")) != nil {
		os.Exit(42)
	}
	fmt.Println("codex-cli 0.166.0")
	os.Exit(0)
}

func TestInspectionReportsCleanupFailureAndPreservesReplacement(t *testing.T) {
	root := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Home: filepath.Join(root, "home"), Scratch: filepath.Join(root, "scratch"), Program: codexrpc.Program{Path: exe, Args: []string{"-test.run=^TestStageReplacementProcess$", "--", "replace-stage"}}}
	_, err = a.Inspect(context.Background())
	if err == nil {
		t.Error("inspection silently accepted failed ownership cleanup")
	} else if strings.Contains(err.Error(), root) {
		t.Fatal("cleanup error leaked private staging path")
	}
	paths, globErr := filepath.Glob(filepath.Join(a.Scratch, "codex-stage-*", "foreign"))
	if globErr != nil || len(paths) != 1 {
		t.Fatal("cleanup removed a foreign replacement directory", globErr)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil || string(data) != "preserve" {
		t.Fatal("foreign replacement changed", err)
	}
}
