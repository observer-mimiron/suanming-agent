package handler

import "testing"

func TestResolveSessionID_EmptyFallsBackToDefault(t *testing.T) {
	got, err := resolveSessionID("")
	if err != nil {
		t.Fatalf("resolveSessionID(\"\") error = %v", err)
	}
	if got != "default" {
		t.Fatalf("resolveSessionID(\"\") = %q, want default", got)
	}
}

func TestResolveSessionID_RejectsUnsafeValues(t *testing.T) {
	cases := []string{
		"../escape",
		`..\escape`,
		"bad/session",
		".",
		"..",
		"bad id",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := resolveSessionID(tc); err == nil {
				t.Fatalf("resolveSessionID(%q) error = nil, want invalid session id", tc)
			}
		})
	}
}
