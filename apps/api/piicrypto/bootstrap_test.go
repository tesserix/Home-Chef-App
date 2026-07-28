package piicrypto

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// InitIfEnabled with the flag off must be a pure no-op — no Secret Manager or
// KMS call (neither is reachable from a unit test, so a non-nil error here would
// also mean it tried), and encryption left inactive.
func TestInitIfEnabled_DisabledIsNoOp(t *testing.T) {
	if err := InitIfEnabled(context.Background(), false, "any-project"); err != nil {
		t.Fatalf("disabled InitIfEnabled should no-op, got %v", err)
	}
	if Active() {
		t.Fatal("disabled InitIfEnabled must leave encryption inactive")
	}
}

// The regression guard for the outage this package caused: PII_ENCRYPTION_ENABLED
// was true in prod, but ONLY apps/api/main.go initialized the key. The Temporal
// worker — a full app process whose activities GORM-scan orders — never did, so
// every activity that loaded a row with an encrypted column failed permanently
// with "encrypted value but crypto not initialized" and Temporal retried it
// forever. A delivery dispatch burned 50+ attempts, and the confirm-receipt
// reminder + auto-confirm flow could never run.
//
// Any binary that talks to the database must initialize PII crypto at startup.
// This asserts it structurally rather than by comment, so a new entrypoint that
// forgets fails here instead of in production.
func TestEveryEntrypointInitializesPIICrypto(t *testing.T) {
	entrypoints := map[string]string{
		"API server":      "../main.go",
		"Temporal worker": "../cmd/worker/main.go",
	}

	for name, path := range entrypoints {
		t.Run(name, func(t *testing.T) {
			src, err := parser.ParseFile(token.NewFileSet(), filepath.Clean(path), nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}

			var found bool
			ast.Inspect(src, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "piicrypto" {
					return true
				}
				if strings.HasPrefix(sel.Sel.Name, "Init") {
					found = true
					return false
				}
				return true
			})

			if !found {
				t.Errorf(
					"%s (%s) never calls piicrypto.Init*: with PII encryption enabled "+
						"every DB read of an encrypted column will fail at runtime. "+
						"Call piicrypto.InitIfEnabled at startup.",
					name, path,
				)
			}
		})
	}
}
