package traverse

import (
	"errors"
	"testing"
)

func TestErrUnauthorized(t *testing.T) {
	if !errors.Is(ErrUnauthorized, ErrUnauthorized) {
		t.Error("ErrUnauthorized should be comparable")
	}
	if ErrUnauthorized.Error() != "traverse: authentication required (401)" {
		t.Errorf("unexpected error message: %s", ErrUnauthorized.Error())
	}
}

func TestErrForbidden(t *testing.T) {
	if !errors.Is(ErrForbidden, ErrForbidden) {
		t.Error("ErrForbidden should be comparable")
	}
	if ErrForbidden.Error() != "traverse: authorization denied (403)" {
		t.Errorf("unexpected error message: %s", ErrForbidden.Error())
	}
}

func TestErrBatchFailed(t *testing.T) {
	if !errors.Is(ErrBatchFailed, ErrBatchFailed) {
		t.Error("ErrBatchFailed should be comparable")
	}
	if ErrBatchFailed.Error() != "traverse: batch request failed" {
		t.Errorf("unexpected error message: %s", ErrBatchFailed.Error())
	}
}

func TestBatchErrorWrapsErrBatchFailed(t *testing.T) {
	err := errors.New("batch failed with status 500")
	wrapped := errors.Join(ErrBatchFailed, err)
	if !errors.Is(wrapped, ErrBatchFailed) {
		t.Error("wrapped error should be ErrBatchFailed")
	}
}
