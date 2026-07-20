package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// P1 is a dual-write phase: the plaintext columns stay authoritative and the
// companions mirror them. With PII_ENCRYPTION_ENABLED off (the default, and the
// state these tests run in) EncryptPII is a pass-through, so the companions hold
// plaintext — which is exactly what makes P1 reversible.

func TestUserBeforeSave_MirrorsEveryPIIColumn(t *testing.T) {
	u := &User{
		Email:     "Chef@Example.com",
		FirstName: "Asha",
		LastName:  "Rao",
		Phone:     "+919876543210",
	}
	require.NoError(t, u.BeforeSave(nil))

	require.Equal(t, "Chef@Example.com", string(u.EmailEnc), "email must be mirrored verbatim")
	require.Equal(t, "Asha", string(u.FirstNameEnc))
	require.Equal(t, "Rao", string(u.LastNameEnc))
	require.Equal(t, "+919876543210", string(u.PhoneEnc))
}

// The blind index must not leak plaintext — that is the whole point of hashing
// it rather than storing a second copy.
func TestUserBeforeSave_BlindIndexIsNeverPlaintext(t *testing.T) {
	u := &User{Email: "chef@example.com", Phone: "+919876543210"}
	require.NoError(t, u.BeforeSave(nil))

	require.NotEqual(t, u.Email, u.EmailBidx)
	require.NotEqual(t, u.Phone, u.PhoneBidx)
}

// An absent optional field must not produce a blind index, or every user with no
// phone would collide on the same value once it carries a unique constraint.
func TestUserBeforeSave_EmptyFieldYieldsNoBlindIndex(t *testing.T) {
	u := &User{Email: "chef@example.com"} // no phone
	require.NoError(t, u.BeforeSave(nil))
	require.Empty(t, u.PhoneBidx, "an absent phone must not get a blind index")
}

func TestAddressBeforeSave_MirrorsLines(t *testing.T) {
	a := &Address{Line1: "12 Residency Rd", Line2: "Apt 4"}
	require.NoError(t, a.BeforeSave(nil))
	require.Equal(t, "12 Residency Rd", string(a.Line1Enc))
	require.Equal(t, "Apt 4", string(a.Line2Enc))
}

// The map-based Updates path bypasses hooks entirely, so PIIUpdates is the only
// thing standing between those call sites and a plaintext write into an
// encrypted column. Guard its shape.
func TestPIIUpdates_EmitsCompanionsForSearchableAndPlainColumns(t *testing.T) {
	got := PIIUpdates(map[string]string{
		"phone":      "+919876543210",
		"first_name": "Asha",
	})

	require.Contains(t, got, "phone_enc", "searchable column needs its ciphertext companion")
	require.Contains(t, got, "phone_bidx", "searchable column needs its blind index")
	require.Contains(t, got, "first_name_enc")
	require.NotContains(t, got, "first_name_bidx", "names are not searched — no blind index")
}

// A column we do not know about must be ignored rather than guessed at, so a
// typo cannot silently write a stray column.
func TestPIIUpdates_IgnoresUnknownColumns(t *testing.T) {
	got := PIIUpdates(map[string]string{"not_a_pii_column": "x"})
	require.Empty(t, got)
}

// P1 must be a behavioural no-op while PII_ENCRYPTION_ENABLED is off: the
// companions mirror plaintext verbatim and the blind index stays empty, so
// nothing reading plaintext today can observe a difference. This is what makes
// the phase reversible — dropping the columns would lose nothing.
func TestP1_IsANoOpWhileEncryptionDisabled(t *testing.T) {
	u := &User{Email: "a@b.com", FirstName: "A", LastName: "B", Phone: "+911234567890"}
	require.NoError(t, u.BeforeSave(nil))

	require.Equal(t, u.Email, string(u.EmailEnc), "companion must mirror plaintext while disabled")
	require.Equal(t, u.Phone, string(u.PhoneEnc))
	require.Empty(t, u.EmailBidx, "no blind index without a key")
	require.Empty(t, u.PhoneBidx)
}
