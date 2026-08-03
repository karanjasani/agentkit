package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
)

func TestSymbolConstBody(t *testing.T) {
	a := newFixtureAnalyzer(t)
	sym, err := a.Symbol(context.Background(), "widgetAPI", api.SymbolOptions{Body: true})
	if err != nil {
		t.Fatal(err)
	}
	if sym.Kind != "const" {
		t.Errorf("kind = %q, want const", sym.Kind)
	}
	if !strings.Contains(sym.Body, "widgetAPI") {
		t.Errorf("body missing const name: %q", sym.Body)
	}
}

func TestSymbolTypeBody(t *testing.T) {
	a := newFixtureAnalyzer(t)
	sym, err := a.Symbol(context.Background(), "models.Widget", api.SymbolOptions{Body: true})
	if err != nil {
		t.Fatal(err)
	}
	if sym.Kind != "type" {
		t.Errorf("kind = %q, want type", sym.Kind)
	}
	if !strings.Contains(sym.Signature, "struct") {
		t.Errorf("type signature should mention struct: %q", sym.Signature)
	}
	if !strings.Contains(sym.Body, "Widget") {
		t.Errorf("type body missing name: %q", sym.Body)
	}
}

func TestSymbolMethod(t *testing.T) {
	a := newFixtureAnalyzer(t)
	sym, err := a.Symbol(context.Background(), "Owner.Display", api.SymbolOptions{Body: true})
	if err != nil {
		t.Fatal(err)
	}
	if sym.Kind != "method" {
		t.Errorf("kind = %q, want method", sym.Kind)
	}
	if sym.Recv != "Owner" {
		t.Errorf("recv = %q, want Owner", sym.Recv)
	}
	if !strings.Contains(sym.Signature, "Display") {
		t.Errorf("signature missing method name: %q", sym.Signature)
	}
}

func TestCallersOfMethod(t *testing.T) {
	a := newFixtureAnalyzer(t)
	// Display has no callers; this exercises the method-resolution path in the
	// call-graph builder and must not error.
	c, err := a.Callers(context.Background(), "Owner.Display")
	if err != nil {
		t.Fatal(err)
	}
	if c.Symbol != "Display" {
		t.Errorf("symbol = %q, want Display", c.Symbol)
	}
}
