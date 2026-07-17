package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// PromptUsername reads a non-empty username from in (or interactive stdin).
func PromptUsername(in io.Reader, out io.Writer) (string, error) {
	if _, err := fmt.Fprint(out, "Username: "); err != nil {
		return "", err
	}
	sc := bufio.NewScanner(in)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", errors.New("username required")
	}
	u := strings.TrimSpace(sc.Text())
	if u == "" {
		return "", errors.New("username required")
	}
	return u, nil
}

// PromptPassword reads a password without echo when stdin is a TTY.
func PromptPassword(out io.Writer) (string, error) {
	if _, err := fmt.Fprint(out, "Password: "); err != nil {
		return "", err
	}
	fd := int(syscall.Stdin)
	if !term.IsTerminal(fd) {
		// non-TTY: read a line (scripts may pipe password)
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return "", err
			}
			return "", errors.New("password required")
		}
		fmt.Fprintln(out)
		return sc.Text(), nil
	}
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(out)
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", errors.New("password required")
	}
	return string(b), nil
}
