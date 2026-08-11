package cli

import (
	"bytes"
	"strings"
	"testing"
)

// CLI 根契约测试（package cli，经 New/Run 组装最终命令树）。
//
// 这是根目录唯一的测试文件，锁定 CLI 稳定的对外行为：根 help 全量字面、命令集合与显示
// 顺序、persistent flags、关键命令组在最终树中的完整 help 字面，以及无网络前置参数错误
// （stderr 文本、stdout 空值、退出码）。命令专属的 flag/前置校验由各 commands/* owner
// 测试拥有，不在此重复。

const rootHelpLiteral = `JavDB app API command-line client

Usage:
  javdb [command]

Available Commands:
  actor       List movies for an actor (id or name)
  auth        Account login and multi-account management
  browse      Browse movies by content tags / year / month
  code        List movies for a code/prefix e.g. SSIS
  collections List a collection: actors|series|codes|makers|directors
  comments    List one page of movie reviews
  completion  Generate the autocompletion script for the specified shell
  config      Show or edit config.toml
  detail      Show movie detail (graph ids for agent navigation)
  director    List movies for a director (id or name)
  download    Download selected movie media to new files
  help        Help about any command
  list        List movies inside a 合集 (user playlist)
  lists       My 合集; subcommands: show/search/related
  magnets     List magnet links for a movie
  maker       List movies for a maker/studio (id or name)
  mark        Mark a movie as 看過 (--watched) or 想看 (--want)
  rankings    Movie/actor rankings (no login needed)
  recent      List recently viewed (最近浏览) movies
  search      Search movies (or other dimensions with --type)
  series      List movies for a series (id or name)
  tags        List content-tag taxonomy (id + EN + 中文)
  top250      TOP250 list (needs login)
  unmark      Remove watched/want mark for a movie
  update      Check for or install updates
  version     Print version
  want        List want-to-watch (想看) movies
  watched     List watched (看過) movies

Flags:
  -h, --help           help for javdb
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)

Use "javdb [command] --help" for more information about a command.
`

func TestRootHelpFullLiteral(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errb.String())
	}
	if out.String() != rootHelpLiteral {
		t.Fatalf("root help mismatch:\n--- want ---\n%q\n--- got ---\n%q", rootHelpLiteral, out.String())
	}
}

// 关键子命令 help 全量字面锁定。cobra 的 help 输出按命令名排序，注册顺序不直接可见，
// 但最终命令树（Task 11 的 root.go）必须产出与这里完全相同的文本。
func TestKeySubcommandHelpLiterals(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"version", `Print version

Usage:
  javdb version [flags]

Flags:
  -h, --help   help for version
      --json   Machine-readable JSON

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"update", `Check for or install updates

Usage:
  javdb update [flags]

Examples:
javdb update --check

Flags:
      --check        Check for an update without installing it
  -h, --help         help for update
      --json         Print update check status as JSON (requires --check)
      --prerelease   Include prerelease updates

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"search", `Search movies (or other dimensions with --type)

Usage:
  javdb search KEYWORD [flags]

Flags:
      --filter-by string   can_play|magnets|subtitle|single
      --has-magnets        Drop movie rows with magnets_count == 0
  -h, --help               help for search
      --json               Machine-readable JSON
      --limit int          Page size (0 = server default)
      --page int           Page number (default 1)
      --sort string        relevance|release|score|update|hit
      --type string        movie|code|series|actor|maker|director|list
      --zone string        censored|uncensored|western|fc2|all (default "censored")

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"detail", `Show movie detail (graph ids for agent navigation)

Usage:
  javdb detail NUMBER [flags]

Flags:
  -h, --help      help for detail
  -i, --id        Treat argument as internal movie id
      --json      Machine-readable JSON
      --magnets   Also list magnet links

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"magnets", `List magnet links for a movie

Usage:
  javdb magnets NUMBER [flags]

Flags:
      --best              Pick single best magnet (cnsub > hd > size)
      --cnsub             Only magnets with Chinese subtitles
      --hd                Only HD magnets
  -h, --help              help for magnets
  -i, --id                Treat NUMBER as internal movie id
      --json              Machine-readable JSON
      --min-size string   Min size e.g. 2000, 4GB, 500MB

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"mark", `Mark a movie as 看過 (--watched) or 想看 (--want)

Usage:
  javdb mark NUMBER [flags]

Flags:
      --content string   Optional review text
  -h, --help             help for mark
  -i, --id               Treat NUMBER as internal movie id
      --score int        Optional score
      --want             Mark as 想看
      --watched          Mark as 看過

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)
`},
		{"lists", `My 合集; subcommands: show/search/related

Usage:
  javdb lists [flags]
  javdb lists [command]

Available Commands:
  related     Public 合集 related to a movie
  search      Search public 合集
  show        Show 合集 meta (movies: use list <id>)

Flags:
  -h, --help             help for lists
      --json             JSON output
      --limit int        Page size (default 20)
      --page int         Page (default 1)
      --sort-by string   created|name|movies_count|views_count|updated|default (default "created")

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)

Use "javdb lists [command] --help" for more information about a command.
`},
		{"rankings", `Movie/actor rankings (no login needed)

Usage:
  javdb rankings [command]

Available Commands:
  actors      Actor rankings
  movies      Movie rankings
  playback    Playback rankings

Flags:
  -h, --help   help for rankings

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)

Use "javdb rankings [command] --help" for more information about a command.
`},
		{"auth", `Account login and multi-account management

Usage:
  javdb auth [command]

Available Commands:
  check       Check default account token (does not print token)
  list        List saved accounts
  login       Log in with username/password (interactive if flags omitted)
  remove      Remove a saved account
  use         Set the default account

Flags:
  -h, --help   help for auth

Global Flags:
      --host string    mirror|main (default: config or mirror)
      --proxy string   Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)

Use "javdb auth [command] --help" for more information about a command.
`},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		code := Run([]string{tc.cmd, "--help"}, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%s --help: code=%d err=%s", tc.cmd, code, errb.String())
		}
		if out.String() != tc.want {
			t.Fatalf("%s help mismatch:\n--- want ---\n%q\n--- got ---\n%q", tc.cmd, tc.want, out.String())
		}
	}
}

// Persistent flags 必须保持 --host/--proxy 的默认值与 usage（FlagSet 层面，与 help 文本一致）。
func TestPersistentFlagsLocked(t *testing.T) {
	root := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	proxy := root.PersistentFlags().Lookup("proxy")
	if proxy == nil {
		t.Fatal("missing --proxy")
	}
	if proxy.DefValue != "" || proxy.Usage != "Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)" {
		t.Fatalf("proxy flag = def %q usage %q", proxy.DefValue, proxy.Usage)
	}
	host := root.PersistentFlags().Lookup("host")
	if host == nil {
		t.Fatal("missing --host")
	}
	if host.DefValue != "" || host.Usage != "mirror|main (default: config or mirror)" {
		t.Fatalf("host flag = def %q usage %q", host.DefValue, host.Usage)
	}
}

// 无网络前置参数错误：任何远程请求发生前就必须失败并给出确定文本与 exit code。
func TestNoNetworkParameterErrorsExact(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"search"}, "accepts 1 arg(s), received 0"},
		{[]string{"detail"}, "accepts 1 arg(s), received 0"},
		{[]string{"mark"}, "accepts 1 arg(s), received 0"},
		{[]string{"update", "--json"}, "--json is only supported with --check"},
		{[]string{"frobnicate"}, `unknown command "frobnicate" for "javdb"`},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		code := Run(tc.args, strings.NewReader(""), &out, &errb)
		if code == 0 {
			t.Fatalf("%v unexpectedly succeeded (out=%q)", tc.args, out.String())
		}
		if errb.String() != tc.want+"\n" {
			t.Fatalf("%v: stderr=%q, want %q", tc.args, errb.String(), tc.want+"\n")
		}
		if out.String() != "" {
			t.Fatalf("%v: stdout=%q, want empty", tc.args, out.String())
		}
	}
}

// 命令集合锁定：最终命令树必须包含全部真实命令（存在性由 root/root_test.go 已覆盖，
// 这里用 help 全量文本覆盖集合与显示顺序）。
func TestRootCommandSetMatchesHelp(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	for _, name := range []string{
		"actor", "auth", "browse", "code", "collections", "comments", "config", "detail",
		"director", "download", "list", "lists", "magnets", "maker", "mark", "rankings",
		"recent", "search", "series", "tags", "top250", "unmark", "update", "version", "want", "watched",
	} {
		if !strings.Contains(out.String(), "  "+name+" ") {
			t.Fatalf("root help missing command %q", name)
		}
	}
}
