package prompt

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

func TestReadMasked(t *testing.T) {
	for _, tc := range []struct {
		name, typed, want, echo string
	}{
		{"enter", "secret\r", "secret", "******"},
		{"backspace", "ab\x7fc\r", "ac", "**\b \b*"},
		{"backspace on empty", "\x08x\n", "x", "*"},
		{"control characters", "a\x1bb\r", "ab", "**"},
		{"unicode", "пароль\r", "пароль", "******"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var echo strings.Builder
			got, err := readMasked(bufio.NewReader(strings.NewReader(tc.typed)), &echo)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want || echo.String() != tc.echo {
				t.Errorf("readMasked() = %q echoing %q, want %q echoing %q", got, echo.String(), tc.want, tc.echo)
			}
		})
	}
}

func TestReadMaskedInterrupted(t *testing.T) {
	_, err := readMasked(bufio.NewReader(strings.NewReader("ab\x03")), &strings.Builder{})
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("readMasked() error = %v, want ErrInterrupted", err)
	}
}

func TestReadMaskedLeavesNextLine(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("pasted\r\nnext\n"))
	if _, err := readMasked(r, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if next, _ := r.ReadString('\n'); next != "next\n" {
		t.Errorf("next line = %q, want %q", next, "next\n")
	}
}
