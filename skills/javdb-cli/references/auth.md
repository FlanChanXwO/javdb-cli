# Authentication and Accounts

The local `~/.javdb-cli/auth.json` contains passwords and JWTs; supported POSIX systems use `0600`. Never read, display, upload, or include it in diagnostics.

For an explicit login request, let the user run `javdb auth login` in a private interactive terminal. Missing username/password arguments trigger prompts. Start it through an agent only when the user can actually type into that terminal; otherwise report the interaction limitation rather than leaving a waiting process.

If the username is already known, `javdb auth login -u USER` may be used, but always omit `-p`/`--password` and enter the password through the hidden interactive prompt. Never transmit passwords through command arguments, tool calls, logs, or results.

`javdb auth list` omits tokens and is appropriate only for an account decision or explicit request. `javdb auth check --json` makes a real API request. A failed check does not authorize automatic login or account/host changes.

`auth use USER_ID` and `auth remove USER_ID` change local account state; require the current explicit target/action. `auto_relogin=true` uses the saved password to relogin once when the selected account JWT expires. Enable it through `javdb config set auto_relogin true` only after explaining that stored-password use and receiving authorization.

Inspect command status before parsing successful JSON. Report redacted causes, not credential values or raw account-store contents.
