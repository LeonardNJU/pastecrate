package main

import "testing"

func TestValidCode(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"ok", true},
		{"lab_42", true},
		{"a-b", true},
		{"a", false},
		{"-ab", false},
		{"create", false},
		{"healthz", false},
		{"space no", false},
		{"../escape", false},
		{"with/slash", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := validCode(tt.code); got != tt.want {
			t.Errorf("validCode(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}
