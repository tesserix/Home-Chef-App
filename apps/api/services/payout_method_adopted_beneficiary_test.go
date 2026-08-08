package services

// #1151. The beneficiary id we compute is a request, not an answer: when the
// account is already registered, Cashfree hands back the id it is registered
// under. A transfer addressed to the computed id would 404, so the id the rail
// returned is the one that must be stored.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

const adoptedMethodsDDL = `CREATE TABLE payout_methods (id TEXT PRIMARY KEY, tenant_id TEXT,
	payee_type TEXT, payee_id TEXT, kind TEXT, status TEXT, "primary" INTEGER DEFAULT 0,
	display_hint TEXT DEFAULT '', beneficiary_name TEXT DEFAULT '', rail TEXT DEFAULT '',
	rail_beneficiary_id TEXT DEFAULT '', rail_status_detail TEXT DEFAULT '',
	verified_at DATETIME, created_at DATETIME, updated_at DATETIME)`

func TestAdoptedBeneficiaryIDIsWhatGetsStored(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(adoptedMethodsDDL).Error)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"conflict_with_existing_beneficiary","message":"already exists"}`))
		case r.URL.Query().Get("bank_account_number") != "":
			_, _ = w.Write([]byte(`{"beneficiary_id":"` + incumbentBeneficiary + `","beneficiary_status":"VERIFIED"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"beneficiary does not exist"}`))
		}
	}))
	t.Cleanup(srv.Close)
	SetCashfreePayoutClientFor(models.ChefModeTest,
		NewCashfreePayoutTestClient(srv.URL, "id", "secret", "", models.ChefModeTest))
	t.Cleanup(func() { SetCashfreePayoutClientFor(models.ChefModeTest, nil) })

	ref := payouts.PayeeRef{Type: payouts.PayeeChef, ID: uuid.New()}
	method, err := EnsurePayoutMethodWith(context.Background(), db, ref, models.ChefModeTest,
		payouts.Instrument{Kind: payouts.MethodBankAccount,
			AccountNumber: conflictAccount, IFSC: conflictIFSC},
		"Saffron Home Kitchen")

	require.NoError(t, err)
	require.Equal(t, incumbentBeneficiary, method.RailBeneficiaryID,
		"a transfer must be addressed to the id the account is actually registered under")
	require.Equal(t, payouts.MethodVerified, method.Status)

	var stored payouts.PayoutMethod
	require.NoError(t, db.First(&stored, "payee_id = ?", ref.ID).Error)
	require.Equal(t, incumbentBeneficiary, stored.RailBeneficiaryID)
}
