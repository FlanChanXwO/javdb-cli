//go:build ignore

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
)

func main() {
	user := strings.TrimSpace(string(mustRead("/tmp/javdb_probe_user")))
	pass := strings.TrimSpace(string(mustRead("/tmp/javdb_probe_pass")))
	c, err := appapi.New(appapi.Options{Host: appapi.HostMirror})
	if err != nil {
		panic(err)
	}
	tok, err := c.Login(user, pass)
	if err != nil {
		fmt.Println("LOGIN_ERR", err)
		os.Exit(1)
	}
	fmt.Println("token_len", len(tok))
	parts := strings.Split(tok, ".")
	if len(parts) >= 2 {
		payload := parts[1]
		switch len(payload) % 4 {
		case 2:
			payload += "=="
		case 3:
			payload += "="
		}
		payload = strings.ReplaceAll(strings.ReplaceAll(payload, "-", "+"), "_", "/")
		b, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			fmt.Println("jwt_decode_err", err)
		} else {
			var claims map[string]any
			_ = json.Unmarshal(b, &claims)
			fmt.Println("jwt_keys", mapKeys(claims))
			for k, v := range claims {
				switch k {
				case "user_id", "uid", "id", "sub", "username", "email", "name":
					fmt.Printf("jwt.%s=%v (%T)\n", k, v, v)
				default:
					fmt.Printf("jwt.%s type=%T\n", k, v)
				}
			}
		}
	}
	if data, err := c.Users(); err != nil {
		fmt.Println("USERS_ERR", err)
	} else {
		fmt.Println("users_keys", rawKeys(data))
		printIDFields("users", data)
		if u, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(u, &nested) == nil {
				fmt.Println("users.user_keys", rawKeys(nested))
				printIDFields("users.user", nested)
			}
		}
	}
	if data, err := c.Startup(); err != nil {
		fmt.Println("STARTUP_ERR", err)
	} else {
		fmt.Println("startup_keys", rawKeys(data))
		if u, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(u, &nested) == nil {
				fmt.Println("startup.user_keys", rawKeys(nested))
				printIDFields("startup.user", nested)
			} else {
				s := string(u)
				if len(s) > 300 {
					s = s[:300] + "…"
				}
				fmt.Println("startup.user raw", s)
			}
		}
	}
	id, name, err := c.ResolveUserID(tok)
	fmt.Println("ResolveUserID", id, name, err)
}

func mustRead(p string) []byte {
	b, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return b
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func rawKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func printIDFields(prefix string, m map[string]json.RawMessage) {
	for _, k := range []string{"id", "user_id", "uid", "username", "email", "name"} {
		if v, ok := m[k]; ok {
			fmt.Printf("%s.%s=%s\n", prefix, k, string(v))
		}
	}
}
