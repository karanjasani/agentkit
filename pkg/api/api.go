// Package api is the public, stable entry point for repository analysis. Both
// the CLI and any future adapter (such as an MCP server) call into this package.
//
// The central type is Analyzer, a handle that loads a module lazily and caches
// the loaded package graph across queries. A long-lived process (like an MCP
// server) can therefore pay the load cost once and answer many queries cheaply.
//
// All methods return typed values from package models and typed *Error values;
// they never return JSON. Applying the transport envelope is the caller's job.
package api

import (
	"context"
	"os"
	"path/filepath"

	"github.com/karanjasani/agentkit/internal/repomap/analyzer"
	"github.com/karanjasani/agentkit/internal/repomap/endpoints"
	"github.com/karanjasani/agentkit/internal/repomap/graph"
	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/pkg/models"
)

// Analyzer is a handle over a single Go module.
type Analyzer struct {
	dir    string
	loader *loader.Loader
}

// Option configures an Analyzer.
type Option func(*config)

type config struct {
	dir string
}

// WithDir sets the module root directory. Defaults to the current directory.
func WithDir(dir string) Option {
	return func(c *config) { c.dir = dir }
}

// New creates an Analyzer for the module rooted at the configured directory.
func New(_ context.Context, opts ...Option) (*Analyzer, error) {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}
	dir := cfg.dir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, newError(CodeInternal, false, "resolving working directory: %v", err)
		}
		dir = wd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, newError(CodeInvalidArgument, true, "invalid directory %q: %v", dir, err)
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return nil, newError(CodeInvalidArgument, true, "not a directory: %s", dir)
	}
	return &Analyzer{dir: abs, loader: loader.New(abs)}, nil
}

// SymbolOptions controls what a Symbol lookup returns.
type SymbolOptions struct {
	Body          bool
	SignatureOnly bool
	Doc           bool
	Shape         bool
}

// ImpactOptions controls change-impact analysis.
type ImpactOptions struct {
	Base string
}

// Overview returns a high-level map of the module.
func (a *Analyzer) Overview(ctx context.Context) (models.Overview, error) {
	return analyzer.Overview(ctx, a.loader)
}

// Package returns information about a single package by import path or directory.
func (a *Analyzer) Package(ctx context.Context, path string) (models.Package, error) {
	return analyzer.PackageInfo(ctx, a.loader, path)
}

// Symbol locates a symbol and returns the requested detail.
func (a *Analyzer) Symbol(ctx context.Context, name string, opts SymbolOptions) (models.Symbol, error) {
	sym, err := analyzer.LookupSymbol(ctx, a.loader, name, analyzer.SymbolDetail{
		Body:          opts.Body,
		SignatureOnly: opts.SignatureOnly,
		Doc:           opts.Doc,
		Shape:         opts.Shape,
	})
	if err != nil {
		return models.Symbol{}, err
	}
	return sym, nil
}

// Struct returns the recursive JSON contract of a struct type.
func (a *Analyzer) Struct(ctx context.Context, name string) (models.Struct, error) {
	return analyzer.StructShape(ctx, a.loader, name)
}

// Deps returns the dependency graph rooted at a package.
func (a *Analyzer) Deps(ctx context.Context, path string) (models.Deps, error) {
	return analyzer.DepsGraph(ctx, a.loader, path)
}

// Callers returns direct and indirect callers of a symbol.
func (a *Analyzer) Callers(ctx context.Context, name string) (models.Callers, error) {
	return graph.Callers(ctx, a.loader, name)
}

// Tests returns tests related to a symbol.
func (a *Analyzer) Tests(ctx context.Context, name string) (models.Tests, error) {
	return graph.RelatedTests(ctx, a.loader, name)
}

// Impact computes the change impact of the working tree against a base ref.
func (a *Analyzer) Impact(ctx context.Context, opts ImpactOptions) (models.Impact, error) {
	return graph.ChangeImpact(ctx, a.loader, a.dir, opts.Base)
}

// Endpoint traces a route through its handler to its upstream calls.
func (a *Analyzer) Endpoint(ctx context.Context, method, path string) (models.Endpoint, error) {
	return endpoints.Trace(ctx, a.loader, method, path)
}

// Upstreams maps outbound calls originating under a package path.
func (a *Analyzer) Upstreams(ctx context.Context, path string) (models.Upstreams, error) {
	return endpoints.UpstreamCalls(ctx, a.loader, path)
}
