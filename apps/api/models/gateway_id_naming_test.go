package models

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// #1119 — the gateway id columns are named gateway_order_id / gateway_payment_id.
// The old name asserted something false about the row: since #933 the CASHFREE
// order id is stamped into it, and payment_provider is what records the rail.
// A missed GORM tag does not fail to compile, it silently reads a column that no
// longer exists, so this guards the rename at the only level that catches it.
var razorpayGatewayID = regexp.MustCompile(
	`RazorpayOrderID|RazorpayPaymentID|RazorpaySignature|` +
		`razorpay_order_id|razorpay_payment_id|razorpay_signature|` +
		`razorpayOrderId|razorpayPaymentId|razorpaySignature`)

// The migration that MOVES the old column has to name it, and so do its tests
// (#1127). Everything else in apps/api must go through gateway_*.
func isGatewayIDNamingExempt(path string) bool {
	for _, exempt := range []string{
		"gateway_id_naming_test.go",
		"database/gateway_id_backfill_test.go",
		"database/gateway_id_backfill_pg_test.go",
	} {
		if strings.HasSuffix(filepath.ToSlash(path), exempt) {
			return true
		}
	}
	return false
}

func TestNoRazorpayNamedGatewayIDs(t *testing.T) {
	root := ".."

	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if isGatewayIDNamingExempt(path) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(body), "\n") {
			if razorpayGatewayID.MatchString(line) {
				offenders = append(offenders, filepath.ToSlash(path)+":"+itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking apps/api: %v", err)
	}

	if len(offenders) > 0 {
		t.Errorf("%d reference(s) still name Razorpay for a gateway id:\n%s",
			len(offenders), strings.Join(offenders, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
