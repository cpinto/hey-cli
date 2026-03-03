# hey-cli

This file provides guidance to AI coding agents working with this repository.

## What is hey-cli?

hey-cli is a CLI and TUI interface for [HEY](https://hey.com). 
It allows users to read and send emails, mange their boxes, manage their calendars and journal entries.
The TUI is primarily intended for human use, while the CLI is primarily intended for use by AI agents and for scripting.

## Development commands

This project uses make.

```bash
make build   # Builds the project into a binary located at ./bin/hey-cli
make test    # Runs the tests
make lint    # Lints the code
make clean   # Cleans the build artifacts
make install # Installs the binary to /usr/local/bin/hey-cli or /usr/bin/hey-cli depending on the system
```

## Architecture Overview

This is a Go project that uses:
- [spf13/cobra](github.com/spf13/cobra) for the CLI interface
- [charm.land/bubbletea/v2] for the TUI interface along with bubbles/v2 and lipgloss/v2 (these are new versions that recently came out and differ from the v1 versions!)

### Authentication

Authentication supports three methods, all managed through `internal/auth/` and `internal/config/`:

1. **OAuth password grant** (primary) — `hey login --client-id ID --client-secret SECRET` prompts for email/password, signs credentials with HMAC-SHA256 (matching Rails MessageVerifier), and exchanges them at `/oauth/tokens` for an access token and refresh token.
2. **Pre-generated bearer token** — `hey login --token TOKEN` stores a token directly.
3. **Browser session cookie** — `hey login --cookie COOKIE` uses an existing HEY.com session.

The API client (`internal/client/`) attaches auth automatically: `Authorization: Bearer <token>` or `Cookie: session_token=<cookie>` (bearer token takes precedence). On a 401 response, it transparently refreshes the access token using the refresh token and retries the request.

All data-access commands call `requireAuth()` before making API calls. `login`, `logout`, and `status` work without authentication.

### State storage

Credentials and config are stored in `~/.config/hey-cli/config.json` with file permissions `0600`. The config includes base URL, access token, refresh token, token expiry (Unix timestamp), OAuth client ID/secret, and install ID. `hey logout` deletes the config file entirely via `config.Clear()`.

## Code style

@STYLE.md
