# jellyfin-download-cli

Download Jellyfin movies and episodes with resumable, sqlite-tracked progress.

## Install

```
go build -o ./dist/jellyfin-download ./
```

## Quick start

```
./dist/jellyfin-download login --server https://jellyfin.example.com --user <username>
./dist/jellyfin-download search "star wars" --type movie
./dist/jellyfin-download download movie --id <itemId>
```

## Series list and interactive search

```
./dist/jellyfin-download series list
./dist/jellyfin-download search --type series --interactive
```

## Speed limit

```
./dist/jellyfin-download download movie --id <itemId> --rate 5M
```

You can also set `JELLYFIN_RATE` for a default rate limit.

## Resume downloads

```
./dist/jellyfin-download downloads resume
```

## Data location

Default store: `~/.jellyfin-download`
- `config.json` — server + token
- `jellyfin.db` — sqlite progress store
- `downloads/` — downloaded media

Override with `--store` or `JELLYFIN_STORE`.
