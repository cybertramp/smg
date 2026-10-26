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
- Deps: all updated; requires Go 1.26+.
