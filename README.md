# jellyfin-download-cli

Download Jellyfin movies and episodes with resumable, sqlite-tracked progress.

## Install

Homebrew:

```
brew install julianfbeck/tap/jellyfin-download
```

From source:

```
go build -o ./dist/jellyfin-download ./
```

## Quick start

```
jellyfin-download login --server https://jellyfin.example.com --user <username>
jellyfin-download search "star wars" --type movie
jellyfin-download download movie --id <itemId>
```

## Series list and interactive search

```
jellyfin-download series list
jellyfin-download search --type series --interactive
```

## Speed limit

```
jellyfin-download download movie --id <itemId> --rate 5M
```

You can also set `JELLYFIN_RATE` for a default rate limit.

## Resume downloads

```
jellyfin-download downloads resume
```

## Data location

Default store: `~/.jellyfin-download`
- `config.json` — server + token
- `jellyfin.db` — sqlite progress store
- `downloads/` — downloaded media

Override with `--store` or `JELLYFIN_STORE`.

## Download layout (Plex-ready)

By default, downloads are structured as:

```
<output>/
  Movie Title (2024)/
    Movie Title (2024).mkv
  Series Name/
    Season 01/
      Series Name - S01E01 - Episode Title.mkv
```

Set the root output directory with `--output`.
