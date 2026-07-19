package models

// encrypted.go — EncryptedString, a GORM column type that transparently encrypts
// PII at rest (#710). Encrypt on write, decrypt on read; behaviour is a no-op
// pass-through until PII_ENCRYPTION_ENABLED is on (piicrypto.Active()), so the
// type can be adopted on a column before the key + migration are live.
//
// Lives in models (not piicrypto) so model structs can use it; it calls down into
// piicrypto, which imports neither models nor services — no import cycle.

import (
	"database/sql/driver"
	"fmt"

	"github.com/homechef/api/piicrypto"
)

// EncryptedString stores its value as AES-GCM ciphertext when encryption is
// enabled, and as plaintext otherwise. Reads transparently decrypt (and pass
// through not-yet-migrated plaintext).
type EncryptedString string

// Value encrypts on the way to the database.
func (e EncryptedString) Value() (driver.Value, error) {
	ct, err := piicrypto.EncryptPII(string(e))
	if err != nil {
		return nil, err
	}
	return ct, nil
}

// Scan decrypts on the way out of the database.
func (e *EncryptedString) Scan(src any) error {
	if src == nil {
		*e = ""
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("EncryptedString: unsupported scan type %T", src)
	}
	pt, err := piicrypto.DecryptPII(s)
	if err != nil {
		return err
	}
	*e = EncryptedString(pt)
	return nil
}

// String returns the plaintext value.
func (e EncryptedString) String() string { return string(e) }
