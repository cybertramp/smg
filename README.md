# smg

**S**imple SSH **M**anager with **G**o(Gopher)

Interactive CLI tool that can connect to SSH

![annoying_gopher](./img.jpg)

## Overview

This program is SSH client program. Can simply add/delete multiple connection information in one place, and can easily manage a lot of SSH connection information from a terminal such as WSL or remote shell.

> I needed a ssh client program like [gossm](https://github.com/gjbae1212/gossm), but gossm is not support custom ssh, so I developed it. This program is a tool that makes ssh management easy and convenient to use.

[![asciicast](https://asciinema.org/a/F8ajcRmlMNBLjrQDnhlyQWaRo.svg)](https://asciinema.org/a/F8ajcRmlMNBLjrQDnhlyQWaRo)

## Run

```bash
smg          # select (type / to search, a: add, e: edit, d: delete) and connect; creates ~/.smg/conn.json on first run
smg <name>   # connect directly (partial name ok; several matches -> list)
smg add      # add item
smg edit     # edit item
smg delete   # delete item
smg version
```

## Authentication (`cert_type`)

| type | meaning |
|------|---------|
| 0 | ssh-agent (`SSH_AUTH_SOCK`) |
| 1 | password. Leave `cert` empty to be prompted at connect (recommended; stored passwords are plaintext) |
| 2 | private key file (`~` allowed). Passphrase is prompted if the key is encrypted |

Host keys are verified against `~/.ssh/known_hosts` (unknown hosts ask for confirmation, changed keys are rejected).

## build 🪟

```bash
# Current platform -> ./dist/smg
go build -o dist/smg

# Linux
GOOS=linux GOARCH=amd64 go build -o dist/smg

# Mac(ARM64)
GOOS=darwin GOARCH=arm64 go build -o dist/smg

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/smg.exe
```

Config file location: `/home/{USERNAME}/.smg/conn.json`

## license

This project following The MIT.

## version history

- 0.1.0
  - fix: `add` saved an empty item
  - security: known_hosts verification, config file 0600 / dir 0700, atomic save
  - change: `init` command removed; config file is created automatically on first run
  - add: `edit`, `delete`, `version`, list filtering, ssh-agent auth, key passphrase prompt, password prompt
  - add: terminal resize, remote exit code propagation, input validation (port/type/duplicate name)
  - refactor: error handling in `main` only, package rename, unit tests, CI (vet/test/govulncheck)
  - update: all dependencies (Go 1.26+)

- 0.04 update features
  - update **help** function
    - This feature show help
  - update **init** function
    - This feature create `~/.smg/conn.json` file
  - update **add** function
    - This feature add item to `~/.smg/conn.json` file
