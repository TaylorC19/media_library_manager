package barcode

import (
	"testing"

	bdom "media_library_manager/internal/domain/barcode"
)

func TestNormalizeBarcode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  978-0-00-000000-0  ", "9780000000000"},
		{"isbn: 0-1-2-x", "012X"},
		{"(abc)1x2M", "1X2"},
	}
	for _, tc := range tests {
		if got := NormalizeBarcode(tc.in); got != tc.want {
			t.Fatalf("NormalizeBarcode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsIsbnLike(t *testing.T) {
	if !IsIsbnLike("9781234567890") {
		t.Fatal("13-digit should be ISBN-like")
	}
	if !IsIsbnLike("0123456789") {
		t.Fatal("10-char should be ISBN-like")
	}
	if IsIsbnLike("123") {
		t.Fatal("short code should not be ISBN-like")
	}
}

func TestBuildStages_BookPreferred(t *testing.T) {
	p := "book"
	h := Hint{PreferredMediaType: &p, IsIsbnLike: true}
	st := BuildStages(h)
	if len(st) != 1 || st[0].MediaType != "book" || st[0].Providers[0] != bdom.ProviderOpenLibrary {
		t.Fatalf("book preferred: got %+v", st)
	}
}

func TestBuildStages_WeakPreferred(t *testing.T) {
	p := "game"
	h := Hint{PreferredMediaType: &p, IsIsbnLike: false}
	if len(BuildStages(h)) != 0 {
		t.Fatal("game preferred should skip provider stages")
	}
}

func TestBuildFallback_NoCandidates_Weak(t *testing.T) {
	weak := "movie"
	m := "movie"
	h := Hint{PreferredMediaType: &weak, IsIsbnLike: false, InferredMediaType: &m}
	f := buildFallback(nil, nil, h, &m, "1234567890123")
	if f == nil || f.Reason != bdom.ReasonWeakBarcodeCoverage {
		t.Fatalf("got %+v", f)
	}
}

func TestBuildFallback_ProviderFailed(t *testing.T) {
	h := Hint{IsIsbnLike: true, InferredMediaType: strPtr2("book")}
	bc := "9780000000000"
	m := "book"
	f := buildFallback(nil, []bdom.Failure{{Provider: "x", Code: bdom.CodeUnavailable}}, h, &m, bc)
	if f == nil || f.Reason != bdom.ReasonProviderUnavailable || f.ManualQuery == nil || *f.ManualQuery != bc {
		t.Fatalf("got %+v", f)
	}
}

func TestBuildFallback_ManualConfirm(t *testing.T) {
	h := Hint{}
	m := "book"
	c := []bdom.Candidate{{Source: "local", MediaType: "book", Title: "Hello"}}
	f := buildFallback(c, nil, h, &m, "x")
	if f == nil || f.Reason != bdom.ReasonManualConfirmation || f.ManualQuery == nil || *f.ManualQuery != "Hello" {
		t.Fatalf("got %+v", f)
	}
}

func strPtr2(s string) *string { return &s }
