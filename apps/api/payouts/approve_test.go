package payouts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecideAutoApprove(t *testing.T) {
	cases := []struct {
		name    string
		in      AutoApproveInput
		approve bool
		reasons []HoldReason
	}{
		{
			name: "clean pass",
			in: AutoApproveInput{
				AutomationEnabled: true, DestinationPaidBefore: true,
				AmountMinor: 100_00, AutoCapMinor: 500_00,
			},
			approve: true,
		},
		{
			name: "automation off",
			in: AutoApproveInput{
				AutomationEnabled: false, DestinationPaidBefore: true,
				AmountMinor: 100_00,
			},
			reasons: []HoldReason{HoldAutomationOff},
		},
		{
			name: "first disbursement to destination",
			in: AutoApproveInput{
				AutomationEnabled: true, DestinationPaidBefore: false,
				AmountMinor: 100_00,
			},
			reasons: []HoldReason{HoldFirstDisbursement},
		},
		{
			name: "above cap",
			in: AutoApproveInput{
				AutomationEnabled: true, DestinationPaidBefore: true,
				AmountMinor: 600_00, AutoCapMinor: 500_00,
			},
			reasons: []HoldReason{HoldAboveAutoCap},
		},
		{
			name: "zero cap disables the check",
			in: AutoApproveInput{
				AutomationEnabled: true, DestinationPaidBefore: true,
				AmountMinor: 99_999_99, AutoCapMinor: 0,
			},
			approve: true,
		},
		{
			name: "unreadable cap fails closed",
			in: AutoApproveInput{
				AutomationEnabled: true, DestinationPaidBefore: true,
				AmountMinor: 1_00, CapUnreadable: true,
			},
			reasons: []HoldReason{HoldCapUnreadable},
		},
		{
			name: "every reason reported, no short-circuit",
			in: AutoApproveInput{
				AutomationEnabled: false, DestinationPaidBefore: false,
				AmountMinor: 600_00, AutoCapMinor: 500_00,
			},
			reasons: []HoldReason{HoldAutomationOff, HoldFirstDisbursement, HoldAboveAutoCap},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := DecideAutoApprove(tc.in)
			require.Equal(t, tc.approve, d.Approve)
			require.Equal(t, tc.reasons, d.Reasons)
		})
	}
}
