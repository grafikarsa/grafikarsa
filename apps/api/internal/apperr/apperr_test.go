package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestAs_FindsWrappedError(t *testing.T) {
	base := NotFound()
	wrapped := fmt.Errorf("load user: %w", base)

	got, ok := As(wrapped)
	if !ok || got.Code != "NOT_FOUND" || got.Status != 404 {
		t.Fatalf("As() = %v, %v; want NOT_FOUND", got, ok)
	}
	if !Is(wrapped, "NOT_FOUND") {
		t.Fatal("Is() = false, want true")
	}
}

func TestAs_PlainErrorIsNotAppErr(t *testing.T) {
	if _, ok := As(errors.New("boom")); ok {
		t.Fatal("As() = true for plain error")
	}
}

func TestWithCause_KeepsCauseAndDoesNotMutateOriginal(t *testing.T) {
	cause := errors.New("db down")
	base := Conflict("username terpakai")
	e := base.WithCause(cause)

	if !errors.Is(e, cause) {
		t.Fatal("cause not unwrapped")
	}
	if base.Cause != nil {
		t.Fatal("original mutated")
	}
}

func TestValidation_HasFields(t *testing.T) {
	e := Validation(Field("title", "too_short", "Judul minimal 3 karakter."))
	if e.Status != 422 || len(e.Fields) != 1 || e.Fields[0].Field != "title" {
		t.Fatalf("unexpected: %+v", e)
	}
}
