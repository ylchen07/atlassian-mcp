package main

import "testing"

func TestEnsureHTTPS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in  string
		out string
	}{
		{"example.atlassian.net", "https://example.atlassian.net"},
		{"https://example.atlassian.net/", "https://example.atlassian.net"},
		{"http://example.atlassian.net", "http://example.atlassian.net"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := ensureHTTPS(tc.in); got != tc.out {
				t.Fatalf("ensureHTTPS(%q) = %q, want %q", tc.in, got, tc.out)
			}
		})
	}
}
