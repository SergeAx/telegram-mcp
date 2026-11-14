// Package prompt asks questions on the terminal, or reads the answers from
// piped input.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var ErrInterrupted = errors.New("interrupted")

// A single reader for every question, so that piped answers are not lost
// in the buffer of a reader created for an earlier one.
var stdin = bufio.NewReader(os.Stdin)

// Interactive reports whether a person can type the answers.
func Interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func Line(question string) (string, error) {
	fmt.Fprint(os.Stderr, question)

	line, err := stdin.ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return "", fmt.Errorf("read answer: %w", err)
	}

	return strings.TrimSpace(line), nil
}

// Masked echoes an asterisk for every character typed.
func Masked(question string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return Line(question)
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("switch terminal to raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, state) }()

	fmt.Fprint(os.Stderr, question)
	answer, err := readMasked(stdin, os.Stderr)
	fmt.Fprint(os.Stderr, "\r\n")

	return answer, err
}

func readMasked(r *bufio.Reader, echo io.Writer) (string, error) {
	var input []rune
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			return "", fmt.Errorf("read answer: %w", err)
		}

		switch c {
		case '\r', '\n':
			if c == '\r' && r.Buffered() > 0 {
				if next, _ := r.Peek(1); next[0] == '\n' {
					_, _ = r.ReadByte()
				}
			}

			return string(input), nil
		case 3, 4: // raw mode delivers Ctrl+C and Ctrl+D as bytes, not signals
			return "", ErrInterrupted
		case 8, 127:
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Fprint(echo, "\b \b")
			}
		default:
			if c >= ' ' {
				input = append(input, c)
				fmt.Fprint(echo, "*")
			}
		}
	}
}
