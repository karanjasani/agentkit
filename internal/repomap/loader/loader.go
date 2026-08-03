// Package loader wraps golang.org/x/tools/go/packages with explicit load-mode
// tiers and per-instance caching. Choosing the minimum load mode per command is
// the primary lever on performance: metadata-only loads are dramatically cheaper
// than full type-checking.
package loader

import (
	"context"
	"fmt"
	"go/token"
	"sync"

	"golang.org/x/tools/go/packages"
)

// Tier selects how much information to load.
type Tier int

const (
	// TierShallow loads names, files, imports and module info only. Suitable for
	// overview and deps. Cheap.
	TierShallow Tier = iota
	// TierTypes loads syntax and full type information for the module and its
	// dependencies. Required for symbol, struct, package exports, callers, etc.
	TierTypes
)

func (t Tier) mode() packages.LoadMode {
	switch t {
	case TierShallow:
		return packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedModule
	default:
		return packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedModule
	}
}

// Result is the outcome of a load: the packages plus shared metadata.
type Result struct {
	Fset       *token.FileSet
	Packages   []*packages.Package
	ModulePath string
	ModuleDir  string
	GoVersion  string
}

// Loader loads a single module rooted at Dir, caching results per (tier, tests).
type Loader struct {
	Dir string

	mu    sync.Mutex
	cache map[cacheKey]*Result
}

type cacheKey struct {
	tier  Tier
	tests bool
}

// New creates a Loader for the module rooted at dir.
func New(dir string) *Loader {
	return &Loader{Dir: dir, cache: make(map[cacheKey]*Result)}
}

// Load loads the whole module ("./...") at the given tier. If tests is true,
// test files and their synthesized packages are included. Results are cached.
func (l *Loader) Load(ctx context.Context, tier Tier, tests bool) (*Result, error) {
	key := cacheKey{tier: tier, tests: tests}

	l.mu.Lock()
	if r, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return r, nil
	}
	l.mu.Unlock()

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode:    tier.mode(),
		Dir:     l.Dir,
		Tests:   tests,
		Context: ctx,
		Fset:    fset,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	res := &Result{Fset: fset, Packages: pkgs}
	for _, p := range pkgs {
		if p.Module != nil && p.Module.Main {
			res.ModulePath = p.Module.Path
			res.ModuleDir = p.Module.Dir
			res.GoVersion = p.Module.GoVersion
			break
		}
	}
	// Fallback if module info was not populated (e.g. GOPATH mode).
	if res.ModuleDir == "" {
		res.ModuleDir = l.Dir
	}

	l.mu.Lock()
	l.cache[key] = res
	l.mu.Unlock()
	return res, nil
}

// FirstError returns the first packages error across the loaded set, if any.
// Type/parse errors are non-fatal for many commands, so callers decide whether
// to surface them.
func FirstError(pkgs []*packages.Package) error {
	var first error
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		if first != nil {
			return
		}
		if len(p.Errors) > 0 {
			first = p.Errors[0]
		}
	})
	return first
}
