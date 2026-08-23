package services

import (
	"context"
	"testing"
)

func TestValidatePublicHTTPSURL_RejectsSSRFAddresses(t *testing.T) {
	t.Parallel()
	tests := []string{
		"http://169.254.169.254/latest/meta-data",
		"https://127.0.0.1",
		"https://10.20.30.40/api",
		"https://[::1]/api",
		"https://user:secret@example.com/api",
		"https://8.8.8.8/api?redirect=https://evil.example",
	}
	for _, rawURL := range tests {
		rawURL := rawURL
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()
			if err := ValidatePublicHTTPSURL(context.Background(), rawURL); err == nil {
				t.Fatalf("ValidatePublicHTTPSURL(%q) succeeded; want rejection", rawURL)
			}
		})
	}
}

func TestValidatePublicHTTPSURL_AllowsPublicHTTPSLiteral(t *testing.T) {
	t.Parallel()
	if err := ValidatePublicHTTPSURL(context.Background(), "https://8.8.8.8/api"); err != nil {
		t.Fatalf("public HTTPS address rejected: %v", err)
	}
}
