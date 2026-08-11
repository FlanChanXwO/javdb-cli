package auth

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

// TestNewBuildsAuthCommandTree 锁定 auth 命令组：顶层 auth + login/list/use/remove/check。
// cobra 的 Command() 返回排序后的子命令，AddCommand 顺序由 root help 契约测试锁定。
func TestNewBuildsAuthCommandTree(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(&invocation.RootOptions{}, streams)
	if command.Name() != "auth" {
		t.Fatalf("command name = %q", command.Name())
	}
	got := map[string]bool{}
	for _, sub := range command.Commands() {
		got[sub.Name()] = true
	}
	for _, name := range []string{"login", "list", "use", "remove", "check"} {
		if !got[name] {
			t.Fatalf("auth missing subcommand %q", name)
		}
	}
}

func TestNewLoginHasExpectedFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(&invocation.RootOptions{}, streams)
	login, _, err := command.Find([]string{"login"})
	if err != nil || login == nil {
		t.Fatalf("find login: %v", err)
	}
	for _, flag := range []string{"username", "password", "use"} {
		if login.Flags().Lookup(flag) == nil {
			t.Fatalf("login missing --%s", flag)
		}
	}
	if login.Flags().Lookup("username").Shorthand != "u" || login.Flags().Lookup("password").Shorthand != "p" {
		t.Fatal("login shorthand changed")
	}
}
