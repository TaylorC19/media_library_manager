package providererrors

import (
	"errors"
	"testing"
)

func TestErrorWrapsCause(t *testing.T) {
	cause := errors.New("boom")
	err := New(Options{
		Provider:  "musicbrainz",
		Operation: "search",
		Code:      CodeRateLimited,
		Message:   "Provider rate limit was reached.",
		Retryable: true,
		Cause:     cause,
	})

	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatal("expected errors.As to match provider error")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped cause to be discoverable")
	}
	if got, want := perr.Error(), "musicbrainz search: Provider rate limit was reached."; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !perr.Retryable {
		t.Fatal("expected retryable error")
	}
}

func TestHelperConstructors(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		code Code
	}{
		{
			name: "configuration",
			err:  Configuration("tmdb", "detail", "TMDB key missing"),
			code: CodeConfiguration,
		},
		{
			name: "invalid_response",
			err:  InvalidResponse("open_library", "search", ""),
			code: CodeInvalidResponse,
		},
		{
			name: "unsupported",
			err:  Unsupported("rawg", "barcode", ""),
			code: CodeUnsupported,
		},
	}

	for _, tc := range cases {
		if tc.err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if tc.err.Code != tc.code {
			t.Fatalf("%s: code = %q, want %q", tc.name, tc.err.Code, tc.code)
		}
		if tc.err.Message == "" {
			t.Fatalf("%s: expected non-empty message", tc.name)
		}
	}
}

func TestIsRetryableCode(t *testing.T) {
	retryable := []Code{CodeTimeout, CodeNetwork, CodeRateLimited, CodeUpstream}
	for _, code := range retryable {
		if !IsRetryableCode(code) {
			t.Fatalf("expected %q to be retryable", code)
		}
	}

	nonRetryable := []Code{CodeNotFound, CodeInvalidResponse, CodeConfiguration}
	for _, code := range nonRetryable {
		if IsRetryableCode(code) {
			t.Fatalf("expected %q to be non-retryable", code)
		}
	}
}
