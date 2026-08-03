package api_test

import (
	"context"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
	"github.com/karanjasani/agentkit/pkg/models"
)

// TestUpstreamsAcrossClients verifies the broadened upstream detection covers
// stdlib net/http verbs, request construction with a non-GET method, gRPC dials,
// and unknown/custom HTTP clients recognized via a literal URL argument.
func TestUpstreamsAcrossClients(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "go.mod", "module example.com/up\n\ngo 1.24\n")

	writeFile(t, dir, "httpcalls.go", `package up

import (
	"context"
	"net/http"
)

func A() { _, _ = http.Get("https://a.example.com/x") }
func B() { _, _ = http.Post("https://b.example.com/y", "application/json", nil) }
func H() { _, _ = http.Head("https://h.example.com/ping") }
func C(ctx context.Context) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, "https://c.example.com/z", nil)
	_ = req
}
`)

	// A local package literally named "grpc" with a Dial constructor, so the
	// gRPC detector's identifier path fires without a real dependency.
	writeFile(t, dir, "grpc/grpc.go", `package grpc

type ClientConn struct{}

func Dial(target string, opts ...any) (*ClientConn, error) { return &ClientConn{}, nil }
`)
	writeFile(t, dir, "usegrpc.go", `package up

import "example.com/up/grpc"

func D() { _, _ = grpc.Dial("orders:50051") }
`)

	// An unknown custom client: not in the allow-list, so it is recognized only
	// via its literal URL argument and reported with "possible" confidence.
	writeFile(t, dir, "client/client.go", `package client

type C struct{}

func (C) Get(url string) error { return nil }
`)
	writeFile(t, dir, "usecustom.go", `package up

import "example.com/up/client"

func E() { _ = client.C{}.Get("https://custom.example.com/api") }
`)

	a, err := api.New(context.Background(), api.WithDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	res, err := a.Upstreams(context.Background(), ".")
	if err != nil {
		t.Fatalf("upstreams: %v", err)
	}

	type want struct {
		method     string
		url        string
		confidence string
	}
	wants := []want{
		{"GET", "https://a.example.com/x", "direct"},
		{"POST", "https://b.example.com/y", "direct"},
		{"HEAD", "https://h.example.com/ping", "direct"},
		{"PUT", "https://c.example.com/z", "direct"},
		{"GRPC", "orders:50051", "direct"},
		{"GET", "https://custom.example.com/api", "possible"},
	}

	for _, w := range wants {
		if !hasUpstream(res.Calls, w.method, w.url, w.confidence) {
			t.Errorf("missing upstream %s %s (%s)\ngot: %+v", w.method, w.url, w.confidence, res.Calls)
		}
	}
}

func hasUpstream(calls []models.Upstream, method, url, confidence string) bool {
	for _, c := range calls {
		if c.Method == method && c.URL == url && c.Confidence == confidence {
			return true
		}
	}
	return false
}
