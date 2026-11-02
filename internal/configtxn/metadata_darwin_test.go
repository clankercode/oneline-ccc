//go:build darwin

package configtxn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
)

func TestDarwinNativeACLRefused(t *testing.T) {
	// chmod resolves user: qualifiers as usernames, not numeric UIDs. Use
	// the effective UID's native account lookup, independent of $USER.
	owner, err := user.LookupId(strconv.Itoa(os.Geteuid()))
	if err != nil {
		t.Fatal("native ACL fixture account lookup failed", err)
	}
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
			rule := fmt.Sprintf("user:%s:allow read", owner.Username)
			if parentACL {
				target = dir
				rule = fmt.Sprintf("user:%s:allow read,file_inherit,directory_inherit", owner.Username)
			}
			if out, err := exec.Command("/bin/chmod", "+a", rule, target).CombinedOutput(); err != nil {
				t.Fatalf("native ACL fixture failed: %v: %s", err, out)
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
