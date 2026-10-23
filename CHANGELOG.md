# Changelog

## 0.1.0
- Fix: `smg add` saved an empty item (final model was not used).
- Security: known_hosts verification (no more `InsecureIgnoreHostKey`), config 0600 / dir 0700, atomic save.
- Add: `edit`, `delete`, `version` commands; list filtering (`/`); duplicate-name check; port default 22.
- Add: ssh-agent auth (cert_type 0), key passphrase prompt, password prompt when `cert` is empty.
- Fix: no `log.Fatal` while terminal is raw; resize (SIGWINCH) handling; remote exit code propagated.
- Fix: `~` expansion only at path start; config read/parse errors reported.
- Change: cert_type 0 was "no password" (always failed); now ssh-agent.
- Refactor: `tui_main` -> `tuimain`, `tui_add_item` -> `tuiadd`, camelCase, gofmt, tests, CI.
- Deps: all updated; requires Go 1.26+.
