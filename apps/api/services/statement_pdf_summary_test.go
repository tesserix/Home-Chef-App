package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// statement_pdf_summary_test.go — #1092. The settlement summary showed gross,
// commission, GST, TDS and a net payout that silently included adjustments the
// chef could not see. A chef whose transfer was reduced to pay down a debt got a
// statement whose arithmetic did not close.

func labelsOf(lines []statementSummaryLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Label)
	}
	return out
}

func findLine(t *testing.T, lines []statementSummaryLine, label string) statementSummaryLine {
	t.Helper()
	for _, l := range lines {
		if l.Label == label {
			return l
		}
	}
	t.Fatalf("no %q line in %v", label, labelsOf(lines))
	return statementSummaryLine{}
}

func TestStatementSummaryLines_ExplainsACollectedRecovery(t *testing.T) {
	stmt := &models.WeeklyStatement{
		GrossRevenue: 1000, PlatformCommission: 60, CGST: 5.4, SGST: 5.4, TDS: 10,
		RecoveryDeductions: 150, NetPayout: 780,
	}

	lines := statementSummaryLines(stmt, 0.06)

	recovery := findLine(t, lines, "Recovery of outstanding balance")
	require.Equal(t, 150.0, recovery.Amount)
	require.True(t, recovery.Negative, "it comes off the payout — never shown as a credit")
	require.Equal(t, "NET PAYOUT", lines[len(lines)-1].Label, "the payable figure stays last")
}

func TestStatementSummaryLines_OmitsAdjustmentsThatDidNotHappen(t *testing.T) {
	stmt := &models.WeeklyStatement{GrossRevenue: 1000, PlatformCommission: 60, TDS: 10, NetPayout: 930}

	labels := labelsOf(statementSummaryLines(stmt, 0.06))

	require.NotContains(t, labels, "Recovery of outstanding balance")
	require.NotContains(t, labels, "Cancellation fees")
	require.NotContains(t, labels, "Bonuses and rewards")
}

func TestStatementSummaryLines_ShowsPenaltiesAndBonusesOnTheRightSide(t *testing.T) {
	stmt := &models.WeeklyStatement{
		GrossRevenue: 1000, PlatformCommission: 60, TDS: 10,
		PenaltyDeductions: 40, BonusAdditions: 25, NetPayout: 915,
	}

	lines := statementSummaryLines(stmt, 0.06)

	require.True(t, findLine(t, lines, "Cancellation fees").Negative)
	require.False(t, findLine(t, lines, "Bonuses and rewards").Negative, "a credit is not a deduction")
}
