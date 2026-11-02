//go:build darwin

package configtxn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDarwinNativeACLRefused(t *testing.T) {
	for _, parentACL := range []bool{false, true} {
		t.Run(fmt.Sprint(parentACL), func(t *testing.T) {
			d, s := fixture(t)
			dir := filepath.Join(d, "target")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "config")
			put(t, p, "original", 0600)
			w := snapshot(t, p, "ours")
			target := p
			rule := fmt.Sprintf("user:%d allow read", os.Geteuid())
			if parentACL {
				target = dir
				rule = fmt.Sprintf("user:%d allow read,file_inherit,directory_inherit", os.Geteuid())
			}
			if out, err := exec.Command("/bin/chmod", "+a", rule, target).CombinedOutput(); err != nil {
				t.Fatalf("native ACL fixture failed: %v (%d output bytes)", err, len(out))
			}
			if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrUnsafe) {
				t.Fatal("native ACL was not refused", err)
			}
			content(t, p, "original")
			if _, err := os.Lstat(s.Root); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("native ACL refusal mutated state", err)
			}
		})
	}
}
