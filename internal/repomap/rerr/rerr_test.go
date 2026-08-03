package rerr

import (
	"errors"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	e := New(SymbolNotFound, true, "missing %s #%d", "Foo", 3)
	if e.Error() != "missing Foo #3" {
		t.Errorf("Error() = %q", e.Error())
	}
	if e.Code != SymbolNotFound || !e.Recoverable {
		t.Errorf("unexpected fields: %+v", e)
	}
}

func TestAs(t *testing.T) {
	if As(nil) != nil {
		t.Error("As(nil) should be nil")
	}
	typed := New(LoadFailed, false, "boom")
	if As(typed) != typed {
		t.Error("As should return the same *Error instance")
	}
	wrapped := As(errors.New("plain failure"))
	if wrapped == nil || wrapped.Code != Internal {
		t.Errorf("As(plain) = %+v, want Internal", wrapped)
	}
	if wrapped.Message != "plain failure" || wrapped.Recoverable {
		t.Errorf("unexpected wrapped error: %+v", wrapped)
	}
}
