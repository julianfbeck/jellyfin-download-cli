package download

import (
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "Movie: Title/Part 1", want: "Movie_ Title_Part 1"},
		{in: "  ..  ", want: "download"},
		{in: "Good_Name-01.mkv", want: "Good_Name-01.mkv"},
	}

	for _, tc := range cases {
		if got := SanitizeFileName(tc.in); got != tc.want {
			t.Fatalf("SanitizeFileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseRateLimit(t *testing.T) {
	cases := []struct {
		in       string
		expectOK bool
		min      float64
		max      float64
	}{
		{in: "5M", expectOK: true, min: 5 * 1024 * 1024, max: 5*1024*1024 + 1},
		{in: "500K", expectOK: true, min: 500 * 1024, max: 500*1024 + 1},
		{in: "1", expectOK: true, min: 1, max: 2},
		{in: "0", expectOK: false},
		{in: "abc", expectOK: false},
		{in: "10Z", expectOK: false},
	}

	for _, tc := range cases {
		lim, err := ParseRateLimit(tc.in)
		if tc.expectOK {
			if err != nil || lim == nil {
				t.Fatalf("ParseRateLimit(%q) expected ok, got err=%v", tc.in, err)
			}
			got := float64(lim.Limit())
			if got < tc.min || got > tc.max {
				t.Fatalf("ParseRateLimit(%q) limit=%v outside [%v, %v]", tc.in, got, tc.min, tc.max)
			}
		} else if err == nil {
			t.Fatalf("ParseRateLimit(%q) expected error", tc.in)
		}
	}
}
