// Package cli is the Cobra command tree for the javdb binary.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
	"github.com/FlanChanXwO/javdb-cli/internal/config"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	"github.com/FlanChanXwO/javdb-cli/javdb"
)

// Run executes the CLI with the given args (usually os.Args[1:]).
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := newRoot(stdin, stdout, stderr)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type rootFlags struct {
	proxy string
	host  string
}

type appIO struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func newRoot(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	rf := &rootFlags{}
	aio := &appIO{in: stdin, out: stdout, err: stderr}

	root := &cobra.Command{
		Use:           "javdb",
		Short:         "JavDB app API command-line client",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVar(&rf.proxy, "proxy", "", "Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)")
	root.PersistentFlags().StringVar(&rf.host, "host", "", "mirror|main (default: config or mirror)")

	root.AddCommand(newAuthCmd(rf, aio))
	root.AddCommand(newConfigCmd(rf, aio))
	root.AddCommand(newSearchCmd(rf, aio))
	root.AddCommand(newDetailCmd(rf, aio))
	root.AddCommand(newMagnetsCmd(rf, aio))
	root.AddCommand(newTagsCmd(rf, aio))
	root.AddCommand(newBrowseCmd(rf, aio))
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "javdb 0.1.0-dev")
		},
	}
}

func loadRuntime(rf *rootFlags) (config.Runtime, error) {
	path, err := config.ConfigPath()
	if err != nil {
		return config.Runtime{}, err
	}
	file, err := config.LoadFile(path)
	if err != nil {
		return config.Runtime{}, err
	}
	if err := config.ValidateHost(rf.host); err != nil {
		return config.Runtime{}, err
	}
	rt := config.Resolve(file, rf.host, rf.proxy, nil)
	// device uuid
	if rt.DeviceUUID == "" {
		dup, err := config.DeviceUUIDPath()
		if err == nil {
			if id, err := appapi.LoadOrCreateDeviceUUID(dup); err == nil {
				rt.DeviceUUID = id
			}
		}
	}
	return rt, nil
}

func openAuth() (*auth.FileStore, *auth.Store, error) {
	path, err := config.AuthPath()
	if err != nil {
		return nil, nil, err
	}
	if _, err := config.EnsureDir(); err != nil {
		return nil, nil, err
	}
	return auth.Open(path)
}

func newClient(rt config.Runtime, token string) (*javdb.Client, error) {
	return javdb.New(
		javdb.WithHost(rt.BaseURL),
		javdb.WithProxy(rt.Proxy),
		javdb.WithToken(token),
		javdb.WithDeviceUUID(rt.DeviceUUID),
		javdb.WithLang(rt.Lang),
	)
}

func newAuthCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Account login and multi-account management",
	}
	cmd.AddCommand(newAuthLoginCmd(rf, aio))
	cmd.AddCommand(newAuthListCmd(aio))
	cmd.AddCommand(newAuthUseCmd(aio))
	cmd.AddCommand(newAuthRemoveCmd(aio))
	cmd.AddCommand(newAuthCheckCmd(rf, aio))
	return cmd
}

func newAuthLoginCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var user, pass string
	use := true
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with username/password (interactive if flags omitted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			if user == "" {
				user, err = PromptUsername(aio.in, aio.out)
				if err != nil {
					return err
				}
			}
			if pass == "" {
				pass, err = PromptPassword(aio.out)
				if err != nil {
					return err
				}
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			ctx := context.Background()
			token, err := c.Login(ctx, user, pass)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			uid, uname, err := c.ResolveUserID(ctx)
			if err != nil {
				return fmt.Errorf("login ok but user id required: %w", err)
			}
			if uname == "" {
				uname = user
			}
			fs, store, err := openAuth()
			if err != nil {
				return err
			}
			store.Upsert(auth.Account{
				UserID:   uid,
				Username: uname,
				Password: pass,
				Token:    token,
			}, use)
			if err := fs.Commit(store); err != nil {
				return err
			}
			fmt.Fprintf(aio.out, "logged in as %s (id=%d)\n", uname, uid)
			return nil
		},
	}
	cmd.Flags().StringVarP(&user, "username", "u", "", "Username / email")
	cmd.Flags().StringVarP(&pass, "password", "p", "", "Password")
	cmd.Flags().BoolVar(&use, "use", true, "Set as default account after login")
	return cmd
}

func newAuthListCmd(aio *appIO) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, err := openAuth()
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Fprintln(aio.err, "(no accounts)")
				return nil
			}
			for _, a := range store.Accounts {
				mark := " "
				if a.UserID == store.DefaultUserID {
					mark = "*"
				}
				fmt.Fprintf(aio.out, "%s\t%d\t%s\thas_token=%v\n", mark, a.UserID, a.Username, a.Token != "")
			}
			return nil
		},
	}
}

func newAuthUseCmd(aio *appIO) *cobra.Command {
	return &cobra.Command{
		Use:   "use <user_id>",
		Short: "Set the default account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			fs, store, err := openAuth()
			if err != nil {
				return err
			}
			if err := store.Use(uid); err != nil {
				return err
			}
			if err := fs.Commit(store); err != nil {
				return err
			}
			a, _ := store.Get(uid)
			fmt.Fprintf(aio.out, "default account → %s (id=%d)\n", a.Username, uid)
			return nil
		},
	}
}

func newAuthRemoveCmd(aio *appIO) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <user_id>",
		Short: "Remove a saved account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			fs, store, err := openAuth()
			if err != nil {
				return err
			}
			if err := store.Remove(uid); err != nil {
				return err
			}
			if err := fs.Commit(store); err != nil {
				return err
			}
			fmt.Fprintf(aio.out, "removed account id=%d\n", uid)
			return nil
		},
	}
}

func newAuthCheckCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check default account token (does not print token)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			_, store, err := openAuth()
			if err != nil {
				return err
			}
			acc, err := store.Default()
			if err != nil {
				return errors.New("no default account; run: javdb auth login")
			}
			type out struct {
				UserID   int64  `json:"user_id"`
				Username string `json:"username"`
				HasToken bool   `json:"has_token"`
				OK       bool   `json:"ok"`
				Error    string `json:"error,omitempty"`
			}
			result := out{UserID: acc.UserID, Username: acc.Username, HasToken: acc.Token != ""}
			if acc.Token == "" {
				result.OK = false
				result.Error = "no token"
			} else {
				c, err := newClient(rt, acc.Token)
				if err != nil {
					return err
				}
				// lightweight: resolve user id again
				if _, _, err := c.ResolveUserID(context.Background()); err != nil {
					result.OK = false
					result.Error = err.Error()
					var ar *appapi.AuthRequired
					if errors.As(err, &ar) || errors.As(err, new(*javdb.AuthRequired)) {
						result.Error = "token expired or invalid; run: javdb auth login"
					}
				} else {
					result.OK = true
				}
			}
			if asJSON {
				enc := json.NewEncoder(aio.out)
				enc.SetEscapeHTML(false)
				return enc.Encode(result)
			}
			if result.OK {
				fmt.Fprintf(aio.out, "ok\t%d\t%s\n", result.UserID, result.Username)
			} else {
				fmt.Fprintf(aio.out, "fail\t%d\t%s\t%s\n", result.UserID, result.Username, result.Error)
				return errors.New(result.Error)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}

func newConfigCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit config.toml",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.ConfigPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(aio.out, p)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get [key]",
		Short: "Print config (or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			s, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				fmt.Fprintf(aio.out, "host=%s\nhttps_proxy=%s\nauto_relogin=%v\nlang=%s\n",
					s.Host, s.HTTPSProxy, s.AutoRelogin, s.Lang)
				return nil
			}
			switch args[0] {
			case "host":
				fmt.Fprintln(aio.out, s.Host)
			case "https_proxy", "proxy":
				fmt.Fprintln(aio.out, s.HTTPSProxy)
			case "auto_relogin":
				fmt.Fprintln(aio.out, s.AutoRelogin)
			case "lang":
				fmt.Fprintln(aio.out, s.Lang)
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			s, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			switch args[0] {
			case "host":
				if err := config.ValidateHost(args[1]); err != nil {
					return err
				}
				s.Host = args[1]
			case "https_proxy", "proxy":
				s.HTTPSProxy = args[1]
			case "auto_relogin":
				s.AutoRelogin = args[1] == "true" || args[1] == "1" || args[1] == "yes"
			case "lang":
				s.Lang = args[1]
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
			return config.SaveFile(path, s)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a config key to default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			s, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			switch args[0] {
			case "host":
				s.Host = config.HostMirror
			case "https_proxy", "proxy":
				s.HTTPSProxy = ""
			case "auto_relogin":
				s.AutoRelogin = false
			case "lang":
				s.Lang = "en"
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
			return config.SaveFile(path, s)
		},
	})
	return cmd
}

// ensure os is used if needed for future
var _ = os.ErrNotExist
