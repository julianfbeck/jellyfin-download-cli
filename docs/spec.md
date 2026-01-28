# jellyfin-download CLI spec (v0)

## Name
`jellyfin-download`

## One-liner
Download Jellyfin movies and episodes with resumable, sqlite-tracked progress.

## Usage
`jellyfin-download [global flags] <command> [args]`

## Commands
- `login` — Authenticate and store a token in the user store directory.
- `logout` — Remove stored credentials.
- `search` — Search movies/series (non-interactive, script-friendly).
- `select` — Interactive picker for movies/series (prompts if TTY).
- `download movie` — Download a single movie by ID or interactive selection.
- `download series` — Download a whole series or selected seasons/episodes.
- `download episode` — Download specific episode(s) by ID.
- `downloads list` — List tracked downloads and their status.
- `downloads show` — Show a single download record.
- `downloads resume` — Resume queued/failed downloads.

## Global flags
- `-h, --help`
- `--version`
- `-q, --quiet` (suppress non-essential output)
- `-v, --verbose` (more logs)
- `--json` (structured output for list/search)
- `--plain` (stable, line-based output)
- `--no-color`
- `--no-input` (disable prompts)
- `--store DIR` (override store directory; default `~/.jellyfin-download`)
- `--server URL` (override server URL from config)
- `--timeout 30s` (API request timeout)

## I/O contract
- stdout: primary output and `--json`/`--plain` data.
- stderr: diagnostics, warnings, errors.
- prompts only when stdin is a TTY (unless `--no-input`).

## Exit codes
- `0` success
- `1` generic failure
- `2` invalid usage/flags
- `3` not authenticated (login required)
- `4` network/API failure
- `5` download failed

## Config + data
- Store dir default: `~/.jellyfin-download`
  - `config.json` (server URL, user ID, token, default rate)
  - `jellyfin.db` (sqlite progress database)
  - `downloads/` (downloaded media)
- Precedence: flags > env > config.
- Env vars:
  - `JELLYFIN_SERVER`
  - `JELLYFIN_TOKEN`
  - `JELLYFIN_USER_ID`
  - `JELLYFIN_STORE`
  - `JELLYFIN_RATE`

## Safety + interactivity
- No passwords via flags. Use prompt or `--password-stdin`.
- `--no-input` + missing required inputs => error.
- `--dry-run` on download commands prints planned items only.

## Examples
- `jellyfin-download login --server https://jellyfin.example.com --user alice`
- `jellyfin-download search "star wars" --type movie --json`
- `jellyfin-download select --type series`
- `jellyfin-download download series --id <seriesId> --season 1 --episode 1,2,3`
- `jellyfin-download download movie --id <itemId> --rate 5M`
- `jellyfin-download downloads list --plain`
