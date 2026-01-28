package config

import "testing"

func TestNormalizeServerURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "https://jellyfin.xhacker.de/web/#/login.html?serverid=d61c91e2846d446f871c52dc534db09a&url=%2Fhome.html",
			want: "https://jellyfin.xhacker.de",
		},
		{
			in:   "jellyfin.xhacker.de/web/#/login.html?serverid=d61c91e2846d446f871c52dc534db09a&url=%2Fhome.html",
			want: "https://jellyfin.xhacker.de",
		},
		{
			in:   "https://example.com/jellyfin/",
			want: "https://example.com/jellyfin",
		},
		{
			in:   "https://example.com/path?x=1#y",
			want: "https://example.com/path",
		},
		{
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		if got := NormalizeServerURL(tc.in); got != tc.want {
			t.Fatalf("NormalizeServerURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
