package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

// PromptUsername reads a non-empty username from in (or interactive stdin).
func PromptUsername(in io.Reader, out io.Writer) (string, error) {
	if _, err := fmt.Fprint(out, i18n.T("cli.username")); err != nil {
		return "", err
	}
	sc := bufio.NewScanner(in)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", errors.New(i18n.T("cli.username_required"))
	}
	u := strings.TrimSpace(sc.Text())
	if u == "" {
		return "", errors.New(i18n.T("cli.username_required"))
	}
	return u, nil
}

// PromptPassword reads a password without echo when stdin is a TTY.
func PromptPassword(out io.Writer) (string, error) {
	if _, err := fmt.Fprint(out, i18n.T("cli.password_2")); err != nil {
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
			return "", errors.New(i18n.T("cli.password_required"))
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
		return "", errors.New(i18n.T("cli.password_required"))
	}
	return string(b), nil
}
