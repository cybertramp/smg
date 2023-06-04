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
./smg
```

## build 🪟

```bash
# Linux
cd smg
GOOS=linux GOARCH=amd64 go build -o release

# Mac(ARM64)
cd smg
GOOS=darwin GOARCH=arm64 go build -o release

# Windows
cd smg
GOOS=windows GOARCH=amd64 go build -o release
```

Config file location: `/home/{USERNAME}/.smg/conn.json`

## license

This project following The MIT.

## version history

- 0.04 update features
  - update **help** function
    - This feature show help
  - update **init** function
    - This feature create `~/.smg/conn.json` file
  - update **add** function
    - This feature add item to `~/.smg/conn.json` file
