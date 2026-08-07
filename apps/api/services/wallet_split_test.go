package services

import "testing"

// wallet_split_test.go — #141/#1086. The per-account Route transfer split this file
// also covered went with the transfer rail; the cases below keep their original
// inputs and now assert only what survives: the clamp, the capture, and whether a
// gateway charge is needed at all. The chef is settled by Easy Split or the weekly
// statement, so no assertion here can shortchange them.

func TestPlanWalletFunding(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		balance     int
		requested   int
		wantWallet  int
		wantCapture int
		wantFull    bool
	}{
		{
			name:  "no wallet — the customer pays the whole total",
			total: 50000, balance: 0, requested: 0,
			wantWallet: 0, wantCapture: 50000, wantFull: false,
		},
		{
			name:  "partial wallet — capture is the remainder",
			total: 50000, balance: 20000, requested: 8000,
			wantWallet: 8000, wantCapture: 42000, wantFull: false,
		},
		{
			name:  "full wallet — nothing to capture at the gateway",
			total: 48000, balance: 60000, requested: 48000,
			wantWallet: 48000, wantCapture: 0, wantFull: true,
		},
		{
			name:  "requested exceeds balance — clamped to balance",
			total: 50000, balance: 15000, requested: 99999,
			wantWallet: 15000, wantCapture: 35000, wantFull: false,
		},
		{
			name:  "requested exceeds total — clamped to total (full wallet)",
			total: 48000, balance: 99999, requested: 99999,
			wantWallet: 48000, wantCapture: 0, wantFull: true,
		},
		{
			name:  "negative requested — clamped to zero",
			total: 50000, balance: 50000, requested: -500,
			wantWallet: 0, wantCapture: 50000, wantFull: false,
		},
		{
			name:  "zero-total order — full wallet with nothing applied",
			total: 0, balance: 50000, requested: 10000,
			wantWallet: 0, wantCapture: 0, wantFull: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := PlanWalletFunding(tc.total, tc.balance, tc.requested)

			if p.WalletAppliedPaise != tc.wantWallet {
				t.Errorf("wallet applied = %d, want %d", p.WalletAppliedPaise, tc.wantWallet)
			}
			if p.CapturePaise != tc.wantCapture {
				t.Errorf("capture = %d, want %d", p.CapturePaise, tc.wantCapture)
			}
			if p.FullWallet != tc.wantFull {
				t.Errorf("fullWallet = %v, want %v", p.FullWallet, tc.wantFull)
			}

			// The customer must never be charged more than the order, nor the wallet
			// debited for credit that was never captured against.
			if p.WalletAppliedPaise+p.CapturePaise != tc.total {
				t.Errorf("wallet %d + capture %d != total %d", p.WalletAppliedPaise, p.CapturePaise, tc.total)
			}
			if p.CapturePaise < 0 {
				t.Errorf("capture is negative: %d", p.CapturePaise)
			}
		})
	}
}
