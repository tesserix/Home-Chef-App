package services

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// TestUploadFileWithoutInitReturnsError locks the fix for a production erasure
// failure. InitStorage is non-fatal in both entrypoints and cmd/worker never
// called it at all, so storageClient was nil in the worker process. UploadFile
// dereferenced it, and because the account-purge sweeper recovers at the top of
// its scan rather than per account, that panic abandoned every remaining account
// in the batch — no account was ever erased, with one recovered-panic line as the
// only evidence.
//
// A nil client must therefore be an ERROR, never a panic: callers can log it,
// retry it, or carry on, but they cannot survive a panic mid-batch.
func TestUploadFileWithoutInitReturnsError(t *testing.T) {
	prev := storageClient
	storageClient = nil
	t.Cleanup(func() { storageClient = prev })

	_, err := UploadFile(context.Background(), "some-bucket", "some/object.json",
		bytes.NewReader([]byte(`{}`)), "application/json")
	if err == nil {
		t.Fatal("UploadFile with a nil client returned no error — it must not appear to succeed")
	}
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("want ErrStorageUnavailable so callers can branch on it, got %v", err)
	}
}

// TestGenerateSignedURLWithoutInitReturnsError covers the other unguarded
// dereference on the same client, so the same nil cannot panic a request path.
func TestGenerateSignedURLWithoutInitReturnsError(t *testing.T) {
	prev := storageClient
	storageClient = nil
	t.Cleanup(func() { storageClient = prev })

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GenerateSignedURL panicked on a nil client instead of erroring: %v", r)
		}
	}()
	if _, err := GenerateSignedURL(context.Background(), "some/object.json", 0); err == nil {
		t.Fatal("GenerateSignedURL with a nil client returned no error")
	}
}
