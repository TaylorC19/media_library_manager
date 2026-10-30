package library

import "testing"

func TestMediaFormatsAllowAppropriateCopies(t *testing.T) {
	for _, tc := range []struct{ kind, allowed, unrelated string }{{"movie", "blu_ray", "hardcover"}, {"tv", "dvd", "vinyl"}, {"album", "vinyl", "ps5"}, {"book", "paperback", "dvd"}, {"game", "ps5", "hardcover"}} {
		if !IsFormatForMediaType(tc.kind, tc.allowed) || IsFormatForMediaType(tc.kind, tc.unrelated) || !IsFormatForMediaType(tc.kind, "") {
			t.Errorf("incorrect format validation for %s", tc.kind)
		}
		if !IsFormatForMediaType(tc.kind, "digital") || !IsFormatForMediaType(tc.kind, "other") {
			t.Error("digital/other choices must be available")
		}
	}
}
