package services

import (
	"errors"
	"os"
)

var ErrSecretWritesPaused = errors.New("payment detail and gateway credential changes are temporarily paused")

// SecretWritesPaused fails closed on an unrecognized migration-pause setting.
func SecretWritesPaused() bool {
	switch os.Getenv("APP_SECRET_WRITES_PAUSED") {
	case "", "false":
		return false
	default:
		return true
	}
}
