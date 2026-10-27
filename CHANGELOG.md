# Changelog

## 0.1.0
- Fix: `smg add` saved an empty item (final model was not used).
- Security: known_hosts verification (no more `InsecureIgnoreHostKey`), config 0600 / dir 0700, atomic save.
- Change: `init` command removed; config file is created automatically on first run.
- Add: `edit`, `delete`, `version` commands; list filtering (`/`); duplicate-name check; port default 22.
- Add: ssh-agent auth (cert_type 0), key passphrase prompt, password prompt when `cert` is empty.
- Fix: no `log.Fatal` while terminal is raw; resize (SIGWINCH) handling; remote exit code propagated.
- Fix: `~` expansion only at path start; config read/parse errors reported.
- Change: cert_type 0 was "no password" (always failed); now ssh-agent.
- Refactor: `tui_main` -> `tuimain`, `tui_add_item` -> `tuiadd`, camelCase, gofmt, tests, CI.
- Fix: false "HOST KEY CHANGED" when the server offers a different key type than the one in known_hosts.
- Fix: password auth failed on servers allowing only keyboard-interactive; now supported (OTP prompts too).
- Add: keepalive (30s), handshake timeout (10s), SIGHUP handling, UTF-8 locale env (LANG/LC_ALL) forwarding.
- Add: `e` (edit) / `d` (delete, y to confirm) keys in the main list; list reloads after each action.
- Add: `a` (add) key in the main list (also on the empty list); cursor stays on the edited/deleted position.
- Add: `smg <name>` connects directly (exact, or unique partial match; several matches open a filtered list).
- Add: list sorted by last use (`last_used` in conn.json, shown as "5m ago").
- Add: key file auth also tries ssh-agent as fallback.
- Add: pods (nested directories) for connections. `conn.json` gets an optional `pods` tree (`{"name","conn","pods"}`); old files load unchanged.
  - enter opens a pod, backspace goes up, `/` searches every pod, `p` new pod, `e` renames a pod, `d` deletes it with its content, `m` moves a connection, `a` adds to the current pod.
  - add/edit form has a `Pod` field (`prod/web`, created if missing); names are unique per pod.
  - `smg <name>`, `smg edit`, `smg delete` work across all pods.
- Deps: all updated; requires Go 1.26+.
