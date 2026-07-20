// pii_dualwrite.go — P1 of #710.
//
// Every PII column gains two companions:
//
//	<col>_enc   EncryptedString — AES-GCM ciphertext (plaintext pass-through
//	                              while PII_ENCRYPTION_ENABLED is off)
//	<col>_bidx  string          — HMAC-SHA256 of the normalized value, for the
//	                              searchable/unique fields only (email, phone)
//
// Writes go to BOTH the plaintext column and its companions ("dual-write");
// reads still come from plaintext. That keeps P1 fully reversible: with the flag
// off the companions simply mirror plaintext, and nothing reads them yet.
//
// The sync happens in a BeforeSave hook per model rather than at each call site,
// so a new handler that sets User.Email cannot forget it.
//
// LIMITATION — map-based updates bypass this. GORM only invokes driver.Valuer
// for typed struct fields, and a BeforeSave hook cannot see values that live in
// a map rather than the struct:
//
//	db.Model(&u).Updates(map[string]any{"phone": p})   // companions NOT synced
//
// Those call sites must add the companions themselves; PIIUpdates below builds
// the extra map entries so they stay in one place. See handlers/upload.go,
// handlers/customer.go and services/provider.go.

package models

import "github.com/homechef/api/piicrypto"

// encOf returns the encrypted companion for a plaintext value. Assigning the
// plaintext to an EncryptedString is deliberate: the encryption happens in
// EncryptedString.Value() at the driver boundary, so this stays a plain copy
// until the row is actually written.
func encOf(plain string) EncryptedString { return EncryptedString(plain) }

// bidxOf returns the blind index for a searchable/unique value. Empty in, empty
// out — so an absent optional field never collides with another absent one.
// Normalization (lowercase + trim) lives in piicrypto.BlindIndex, which is what
// preserves the existing lower(email) uniqueness semantics.
func bidxOf(plain string) string { return piicrypto.BlindIndex(plain) }

// PIIUpdates returns the companion column entries for a map-based Updates call.
// Pass the plaintext columns being written; the returned map holds only the
// companions, ready to be merged into the caller's update map.
//
//	updates := map[string]any{"phone": p}
//	for k, v := range models.PIIUpdates(map[string]string{"phone": p}) {
//	    updates[k] = v
//	}
//
// Only columns with a known companion shape are emitted; an unknown key is
// ignored rather than guessed at, so a typo cannot silently create a stray
// column write.
func PIIUpdates(plain map[string]string) map[string]any {
	out := make(map[string]any, len(plain)*2)
	for col, v := range plain {
		spec, ok := piiColumns[col]
		if !ok {
			continue
		}
		out[col+"_enc"] = encOf(v)
		if spec.searchable {
			out[col+"_bidx"] = bidxOf(v)
		}
	}
	return out
}

type piiColumnSpec struct{ searchable bool }

// piiColumns is the registry of plaintext PII columns and whether they need a
// blind index. Searchable = looked up by equality or carrying a unique index.
var piiColumns = map[string]piiColumnSpec{
	"email":       {searchable: true},
	"phone":       {searchable: true},
	"first_name":  {},
	"last_name":   {},
	"line1":       {},
	"line2":       {},
	"rider_name":  {},
	"rider_phone": {},
}
