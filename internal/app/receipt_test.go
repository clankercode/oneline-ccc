package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type failedTransaction struct {
	*configtxn.Store
	result configtxn.Result
}

func (s failedTransaction) Apply(context.Context, []configtxn.Write) (configtxn.Result, error) {
	return s.result, errors.New("fixture transaction interruption")
}

func TestFailedApplyReceiptCorrelatesRecoveryTransaction(t *testing.T) {
	for _, txnID := range []string{"fixture-transaction-id", ""} {
		t.Run("transaction-"+txnID, func(t *testing.T) {
			e, _, _, o := fixture(t, false)
			p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
			e.store = failedTransaction{result: configtxn.Result{ID: txnID, Outcome: "Unknown"}}
			r, err := e.Apply(context.Background(), p.ID, p.Revision)
			wantID := txnID
			if wantID == "" {
				wantID = p.ID
			}
			if err == nil || r.ID != wantID || r.Outcome != "Unknown" {
				t.Fatalf("recovery receipt: %#v %v; want ID %s", r, err, wantID)
			}
			if _, err := os.Stat(filepath.Join(o.StateRoot, "settings.json")); !os.IsNotExist(err) {
				t.Fatal("failed transaction persisted attestation")
			}
		})
	}
}
