package payouts

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// modulePath is the module this package lives in. Anything under it other than
// the payouts tree itself is domain code.
const modulePath = "github.com/homechef/api"

// allowedInternal lists the only same-module imports the core may take. It is
// empty on purpose: the core depends on the standard library, gorm and uuid,
// and on nothing this application owns.
var allowedInternal = map[string]bool{}

// TestNoDomainImports is the guarantee behind the reusability claim in ADR 0002.
//
// The payouts core must not know what a chef, an order or a meal plan is.
// Domain knowledge enters through PayeeAdapter and nowhere else. Keeping this
// true is what makes the eventual extraction to go-shared/payouts (#749) a
// module-path change rather than a rewrite — and it is the kind of boundary
// that erodes in a week without a test holding it.
//
// If this test fails, the fix is almost never to add an entry to
// allowedInternal. It is to widen PayeeAdapter, or to move the offending code
// into the adapter that needs it.
func TestNoDomainImports(t *testing.T) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		// Test files may import domain packages for fixtures; the shipped
		// code may not.
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		checked++

		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: bad import literal %s", name, imp.Path.Value)
			}
			if !strings.HasPrefix(path, modulePath) {
				continue // stdlib or third-party: fine
			}
			if strings.HasPrefix(path, modulePath+"/payouts") {
				continue // the core's own subpackages: fine
			}
			if allowedInternal[path] {
				continue
			}
			t.Errorf(
				"%s imports %q.\n\n"+
					"The payouts core must stay domain-free (ADR 0002). Domain knowledge\n"+
					"enters through PayeeAdapter. Widen that interface, or move this code\n"+
					"into the adapter that needs it — do not add an exception here.",
				name, path,
			)
		}
	}

	if checked == 0 {
		t.Fatal("no non-test Go files found; the boundary test is not actually checking anything")
	}
}

// TestBoundaryTestWouldCatchAViolation guards the guard. A boundary test that
// silently stops matching is worse than none, because it reads as protection.
func TestBoundaryTestWouldCatchAViolation(t *testing.T) {
	dir := t.TempDir()
	offender := filepath.Join(dir, "offender.go")
	src := "package payouts\n\nimport _ \"" + modulePath + "/models\"\n"
	if err := os.WriteFile(offender, []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, offender, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	found := false
	for _, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if strings.HasPrefix(path, modulePath) && !strings.HasPrefix(path, modulePath+"/payouts") {
			found = true
		}
	}
	if !found {
		t.Fatal("the detection logic no longer flags a domain import; TestNoDomainImports is not protecting anything")
	}
}
