package services

import "testing"

// sumTransfers totals a transfer slice's paise for the given account, or all
// accounts when account == "".
func sumTransfers(ts []TransferSpec, account string) int {
	total := 0
	for _, t := range ts {
		if account == "" || t.Account == account {
			total += t.Amount
		}
	}
	return total
}

func TestPlanWalletFunding(t *testing.T) {
	chef := "acc_chef"
	driver := "acc_driver"

	tests := []struct {
		name        string
		total       int
		balance     int
		requested   int
		settlements []Settlement
		wantWallet  int
		wantCapture int
		wantFull    bool
		// per-account totals that must hold (payment + topup == owed)
		owed map[string]int
	}{
		{
			name:  "no wallet — all funded from payment, unchanged",
			total: 50000, balance: 0, requested: 0,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  0, wantCapture: 50000, wantFull: false,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "wallet within platform margin — absorbed, no top-up",
			total: 50000, balance: 20000, requested: 8000, // margin = 50000-48000 = 2000... wallet 8000 > margin
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  8000, wantCapture: 42000, wantFull: false,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "wallet exceeds margin — driver topped up from float",
			total: 50000, balance: 50000, requested: 10000,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  10000, wantCapture: 40000, wantFull: false,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "full wallet — no gateway, both topped up",
			total: 48000, balance: 60000, requested: 48000,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  48000, wantCapture: 0, wantFull: true,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "requested exceeds balance — clamped to balance",
			total: 50000, balance: 15000, requested: 99999,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  15000, wantCapture: 35000, wantFull: false,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "requested exceeds total — clamped to total (full wallet)",
			total: 48000, balance: 99999, requested: 99999,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  48000, wantCapture: 0, wantFull: true,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
		{
			name:  "chef FSSAI-withheld (no account) — chef slice stays with platform",
			total: 50000, balance: 50000, requested: 10000,
			settlements: []Settlement{{Account: "", Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  10000, wantCapture: 40000, wantFull: false,
			owed: map[string]int{driver: 6000}, // chef has no account → never transferred
		},
		{
			name:  "negative requested — clamped to zero",
			total: 50000, balance: 50000, requested: -500,
			settlements: []Settlement{{Account: chef, Amount: 42000}, {Account: driver, Amount: 6000}},
			wantWallet:  0, wantCapture: 50000, wantFull: false,
			owed: map[string]int{chef: 42000, driver: 6000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := PlanWalletFunding(tc.total, tc.balance, tc.requested, tc.settlements)

			if p.WalletAppliedPaise != tc.wantWallet {
				t.Errorf("wallet applied = %d, want %d", p.WalletAppliedPaise, tc.wantWallet)
			}
			if p.CapturePaise != tc.wantCapture {
				t.Errorf("capture = %d, want %d", p.CapturePaise, tc.wantCapture)
			}
			if p.FullWallet != tc.wantFull {
				t.Errorf("fullWallet = %v, want %v", p.FullWallet, tc.wantFull)
			}

			// SAFETY INVARIANT 1: payment-linked transfers must never exceed the
			// captured amount (Razorpay rejects otherwise).
			if got := sumTransfers(p.PaymentTransfers, ""); got > p.CapturePaise {
				t.Errorf("payment transfers %d exceed capture %d", got, p.CapturePaise)
			}

			// SAFETY INVARIANT 2: each party is paid in full — payment + top-up
			// equals what they are owed. A bug here would shortchange a chef/driver.
			for acct, want := range tc.owed {
				got := sumTransfers(p.PaymentTransfers, acct) + sumTransfers(p.DirectTopUps, acct)
				if got != want {
					t.Errorf("account %s funded %d, want %d (payment=%d topup=%d)",
						acct, got, want, sumTransfers(p.PaymentTransfers, acct), sumTransfers(p.DirectTopUps, acct))
				}
			}

			// SAFETY INVARIANT 3: a withheld/absent account (owed not asserted)
			// must receive nothing.
			for _, s := range tc.settlements {
				if s.Account == "" {
					continue
				}
				if _, asserted := tc.owed[s.Account]; !asserted {
					if got := sumTransfers(p.PaymentTransfers, s.Account) + sumTransfers(p.DirectTopUps, s.Account); got != 0 {
						t.Errorf("account %s should receive nothing, got %d", s.Account, got)
					}
				}
			}
		})
	}
}
