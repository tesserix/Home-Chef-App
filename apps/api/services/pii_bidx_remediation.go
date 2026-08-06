package services

import (
	"log"
	"regexp"

	"gorm.io/gorm"

	"github.com/homechef/api/piicrypto"
)

// pii_bidx_remediation.go — scrub plaintext phone numbers out of the column that
// was supposed to hold their keyed hash (#925).
//
// handlers/mfa.go wrote services.NormalizeE164(phone) straight into
// phone_e164_bidx. A _bidx column exists so a value can be found WITHOUT being
// stored in a recoverable form; putting the number in the clear one column over
// from its ciphertext made encrypting phone_e164_enc pointless for every user
// who enrolled a phone. The write is fixed at source, but rows written before
// that fix still hold the number, and a fix that leaves the existing exposure in
// place has not actually remediated anything.
//
// This is safe to run because nothing reads phone_e164_bidx yet — it exists for
// #710's P2 read-cutover, which has not happened. So rewriting it cannot break a
// lookup, and it must land BEFORE that cutover, which assumes _bidx columns are
// trustworthy.
//
// Idempotent: a value that already looks like a blind index is left alone, so
// repeated boots re-scan cheaply and converge.

// blindIndexShape matches the output of piicrypto.BlindIndex — hex-encoded
// HMAC-SHA256, so exactly 64 lowercase hex characters. Anything else in a _bidx
// column was not produced by BlindIndex, which for this column means plaintext.
var blindIndexShape = regexp.MustCompile(`^[0-9a-f]{64}$`)

// LooksLikeBlindIndex reports whether a stored _bidx value has the shape
// BlindIndex produces. Exported for the remediation's tests and for any future
// audit of the other _bidx columns.
func LooksLikeBlindIndex(v string) bool { return blindIndexShape.MatchString(v) }

// RemediatePlaintextPhoneBidx re-indexes user_mfa_settings rows whose
// phone_e164_bidx holds something other than a blind index. Returns how many
// rows it rewrote.
//
// Best-effort and non-fatal: a boot must not be blocked by this, and the table
// ships separately (tesserix-k8s), so on a deployment where it does not exist
// yet the query simply fails and we move on.
func RemediatePlaintextPhoneBidx(db *gorm.DB) int {
	if db == nil {
		return 0
	}
	if !db.Migrator().HasTable("user_mfa_settings") {
		return 0
	}

	type row struct {
		UserID        string
		PhoneE164Bidx string
	}
	var rows []row
	if err := db.Table("user_mfa_settings").
		Select("user_id, phone_e164_bidx").
		Where("COALESCE(phone_e164_bidx, '') <> ''").
		Scan(&rows).Error; err != nil {
		log.Printf("pii-bidx remediation: query failed: %v", err)
		return 0
	}

	fixed := 0
	for _, r := range rows {
		if LooksLikeBlindIndex(r.PhoneE164Bidx) {
			continue // already a hash
		}
		// The plaintext IS the stored value, so it can be re-indexed directly —
		// no need to touch the encrypted column. Normalize first, exactly as the
		// fixed write site does, so a row remediated now matches a row written
		// later by the handler.
		// With no blind-index key configured BlindIndex returns "", so this
		// CLEARS the column rather than hashing it. That is the right trade: the
		// point of this sweep is that the number must not sit here in the clear,
		// and nothing reads the column yet. #710's read-cutover has to backfill
		// indexes from the encrypted column regardless — an empty cell is a
		// known-missing index, plaintext is a leak.
		hashed := piicrypto.BlindIndex(NormalizeE164(r.PhoneE164Bidx))
		if err := db.Table("user_mfa_settings").
			Where("user_id = ?", r.UserID).
			Update("phone_e164_bidx", hashed).Error; err != nil {
			log.Printf("pii-bidx remediation: could not re-index user %s: %v", r.UserID, err)
			continue
		}
		fixed++
	}

	if fixed > 0 {
		// Deliberately counts only — never log the value being scrubbed.
		log.Printf("pii-bidx remediation: re-indexed %d phone_e164_bidx row(s) that held plaintext (#925)", fixed)
	}
	return fixed
}
