package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

func TestParseFormat(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Format
		wantErr bool
	}{
		{"json", FormatJSON, false},
		{"text", FormatText, false},
		{"yaml", "", true},
		{"", "", true},
	} {
		got, err := ParseFormat(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseFormat(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseFormat(%q) = %v, %v", tc.in, got, err)
		}
	}
}

func TestSuccessJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	res := models.Overview{Module: "example.com/x", Packages: []models.PkgRef{}}
	if err := Success(&buf, FormatJSON, res); err != nil {
		t.Fatalf("Success: %v", err)
	}
	var env models.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Schema != models.Schema {
		t.Errorf("schema = %q, want %q", env.Schema, models.Schema)
	}
	if !env.Ok {
		t.Error("ok = false, want true")
	}
	if env.ToolVersion == "" {
		t.Error("tool_version empty")
	}
}

func TestSuccessTextDoesNotWrapEnvelope(t *testing.T) {
	var buf bytes.Buffer
	res := models.Overview{Module: "example.com/x", Packages: []models.PkgRef{}}
	if err := Success(&buf, FormatText, res); err != nil {
		t.Fatalf("Success: %v", err)
	}
	if strings.Contains(buf.String(), "\"schema\"") {
		t.Error("text output should not contain JSON envelope")
	}
	if !strings.Contains(buf.String(), "example.com/x") {
		t.Error("text output missing module name")
	}
}

func TestLoc(t *testing.T) {
	if got := loc(models.Location{}); got != "?" {
		t.Errorf("loc(empty) = %q, want ?", got)
	}
	if got := loc(models.Location{File: "a.go", Line: 3}); got != "a.go:3" {
		t.Errorf("loc = %q, want a.go:3", got)
	}
}

func TestFailureText(t *testing.T) {
	var buf bytes.Buffer
	code := Failure(&buf, FormatText, rerr.New(rerr.SymbolNotFound, true, "nope"))
	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
	if !strings.Contains(buf.String(), "error [SYMBOL_NOT_FOUND]: nope") {
		t.Errorf("text error output = %q", buf.String())
	}
}

func TestFailureWrapsPlainError(t *testing.T) {
	var buf bytes.Buffer
	code := Failure(&buf, FormatJSON, errors.New("unexpected"))
	if code != ExitError {
		t.Errorf("exit = %d, want %d (internal)", code, ExitError)
	}
	var env models.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error == nil || env.Error.Code != string(rerr.Internal) {
		t.Errorf("expected INTERNAL error envelope, got %+v", env.Error)
	}
}

func TestFailureExitCodes(t *testing.T) {
	for _, tc := range []struct {
		code rerr.Code
		want int
	}{
		{rerr.SymbolNotFound, ExitNotFound},
		{rerr.PackageNotFound, ExitNotFound},
		{rerr.InvalidArgument, ExitUsageError},
		{rerr.LoadFailed, ExitError},
		{rerr.GitUnavailable, ExitError},
	} {
		var buf bytes.Buffer
		got := Failure(&buf, FormatJSON, rerr.New(tc.code, true, "x"))
		if got != tc.want {
			t.Errorf("Failure(%s) exit = %d, want %d", tc.code, got, tc.want)
		}
		var env models.Envelope
		if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Ok || env.Error == nil || env.Error.Code != string(tc.code) {
			t.Errorf("bad error envelope for %s: %+v", tc.code, env.Error)
		}
	}
}
