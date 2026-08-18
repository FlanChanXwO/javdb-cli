package main

import (
	"strings"
	"testing"
)

// main 只保留入口分派：无参数、help、未知命令与各子命令的分派错误。
// 各子命令的行为测试已迁移到 document/audit/github/history 领域包。

func TestRunRequiresSubcommand(t *testing.T) {
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "subcommand is required") {
		t.Fatalf("run() error = %v, want subcommand required", err)
	}
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		err := run(args)
		if err == nil || !strings.Contains(err.Error(), "usage:") {
			t.Fatalf("run(%v) error = %v, want usage", args, err)
		}
	}
}

func TestRunUnknownSubcommand(t *testing.T) {
	if err := run([]string{"bogus"}); err == nil || !strings.Contains(err.Error(), `unknown subcommand "bogus"`) {
		t.Fatalf("run(bogus) error = %v, want unknown subcommand", err)
	}
}

func TestRunDispatchesEverySubcommand(t *testing.T) {
	for _, name := range []string{"validate", "audit", "render", "sync-history"} {
		err := run([]string{name})
		if err == nil {
			t.Fatalf("run(%s) with no args unexpectedly succeeded", name)
		}
		if strings.Contains(err.Error(), "unknown subcommand") {
			t.Fatalf("run(%s) was not dispatched: %v", name, err)
		}
	}
}
