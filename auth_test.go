package main

import "testing"

func TestMaskPhone(t *testing.T) {
	for in, want := range map[string]string{
		"+79161234567":      "+7.....4567",
		"79161234567":       "+7.....4567",
		"+1 (555) 123-4567": "+1.....4567",
		"+4930123456789":    "+49.....6789",
		"+381641234567":     "+381.....4567",
		"+420601123456":     "+420.....3456",
		"12345":             "+.....",
		"":                  "+.....",
	} {
		if got := maskPhone(in); got != want {
			t.Errorf("maskPhone(%q) = %q, want %q", in, got, want)
		}
	}
}
