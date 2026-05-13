package views

import "testing"

func TestSwapLocalePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		target string
		want   string
	}{
		{name: "empty path", path: "", target: "ja", want: "/ja/"},
		{name: "root path", path: "/", target: "ja", want: "/ja/"},
		{name: "locale root without trailing slash", path: "/en", target: "ja", want: "/ja/"},
		{name: "locale root with trailing slash", path: "/en/", target: "ja", want: "/ja/"},
		{name: "nested locale path", path: "/en/catalog", target: "ja", want: "/ja/catalog"},
		{name: "non locale path", path: "/catalog", target: "ja", want: "/ja/catalog"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := swapLocalePath(tt.path, tt.target); got != tt.want {
				t.Fatalf("swapLocalePath(%q, %q) = %q, want %q", tt.path, tt.target, got, tt.want)
			}
		})
	}
}
