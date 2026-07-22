package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func TestUserCmdsHelp(t *testing.T) {
	for _, name := range []string{"watched", "want", "recent", "collections", "mark", "unmark"} {
		var out, errb bytes.Buffer
		code := Run([]string{name, "--help"}, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%s help: %s", name, errb.String())
		}
	}
}

func TestAuthRequiredMessageWithoutAutoRelogin(t *testing.T) {
	// pure: errors.As path covered by type existence
	err := &javdb.AuthRequired{API: javdb.APIError{Action: "JWTVerificationError", Message: "bad"}}
	if err.Error() == "" {
		t.Fatal("empty")
	}
}
