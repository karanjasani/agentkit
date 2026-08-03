package loader

import (
	"context"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

const fixture = "../../../testdata/fixtures/sample"

func TestFirstError(t *testing.T) {
	if err := FirstError(nil); err != nil {
		t.Errorf("FirstError(nil) = %v, want nil", err)
	}
	clean := &packages.Package{PkgPath: "m/clean"}
	if err := FirstError([]*packages.Package{clean}); err != nil {
		t.Errorf("FirstError(clean) = %v, want nil", err)
	}
	broken := &packages.Package{
		PkgPath: "m/broken",
		Errors:  []packages.Error{{Msg: "boom"}},
	}
	if err := FirstError([]*packages.Package{clean, broken}); err == nil {
		t.Error("FirstError expected an error for a package with Errors")
	}
}

func TestLoadTiersAndCache(t *testing.T) {
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	l := New(abs)
	ctx := context.Background()

	shallow, err := l.Load(ctx, TierShallow, false)
	if err != nil {
		t.Fatalf("shallow load: %v", err)
	}
	if shallow.ModulePath != "example.com/sample" {
		t.Errorf("module path = %q", shallow.ModulePath)
	}
	if len(shallow.Packages) == 0 {
		t.Fatal("no packages loaded")
	}

	// Cached call returns the same pointer.
	again, err := l.Load(ctx, TierShallow, false)
	if err != nil {
		t.Fatal(err)
	}
	if again != shallow {
		t.Error("expected cached result pointer to be reused")
	}

	types, err := l.Load(ctx, TierTypes, false)
	if err != nil {
		t.Fatalf("types load: %v", err)
	}
	if types == shallow {
		t.Error("different tiers must not share cache entries")
	}

	// No packages should carry hard errors for the clean fixture.
	if err := FirstError(types.Packages); err != nil {
		t.Errorf("unexpected package error: %v", err)
	}
}

func TestLoadModeTiers(t *testing.T) {
	if TierShallow.mode() == TierTypes.mode() {
		t.Error("tier modes should differ")
	}
}
