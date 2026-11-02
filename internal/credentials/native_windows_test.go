//go:build windows

package credentials

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// Opt-in only on the disposable hosted runner, never an operator's normal test
// run. A unique service/account avoids touching the application's saved key.
func TestNativeWindowsCredentialManager(t *testing.T) {
	if os.Getenv("ONELINE_TEST_NATIVE_VAULT") != "disposable-ci" || os.Getenv("CI") != "true" {
		t.Skip("requires explicit disposable native CI vault permission")
	}
	service := "oneline-ccc-native-fixture"
	account := fmt.Sprintf("fixture-%d", time.Now().UnixNano())
	key := "fixture-only-not-an-operator-key"
	v := nativeVault{}
	defer func() {
		if err := keyring.Delete(service, account); err != nil {
			t.Error("native fixture cleanup failed")
		}
	}()
	if err := v.Set(service, account, key); err != nil {
		t.Fatal("native credential write failed")
	}
	actual, err := v.Get(service, account)
	if err != nil || actual != key {
		t.Fatal("native credential read did not match")
	}
}
