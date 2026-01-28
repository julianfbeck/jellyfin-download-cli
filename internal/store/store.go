package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName = "jellyfin.db"
)

type Store struct {
	db *sql.DB
}

type Download struct {
	ID            int64
	ItemID        string
	ItemName      string
	ItemType      string
	SeriesID      sql.NullString
	SeasonNumber  sql.NullInt64
	EpisodeNumber sql.NullInt64
	Status        string
	BytesTotal    sql.NullInt64
	BytesDone     sql.NullInt64
	Path          string
	Error         sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func DBPath(storeDir string) string {
	return filepath.Join(storeDir, dbFileName)
}

func Open(storeDir string) (*Store, error) {
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return nil, fmt.Errorf("creating store dir: %w", err)
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", DBPath(storeDir)))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS downloads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	item_id TEXT NOT NULL,
	item_name TEXT NOT NULL,
	item_type TEXT NOT NULL,
	series_id TEXT,
	season_number INTEGER,
	episode_number INTEGER,
	status TEXT NOT NULL,
	bytes_total INTEGER,
	bytes_done INTEGER,
	path TEXT NOT NULL,
	error TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_downloads_item_path ON downloads(item_id, path);
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);

CREATE TABLE IF NOT EXISTS series_progress (
	series_id TEXT PRIMARY KEY,
	last_season INTEGER,
	last_episode INTEGER,
	updated_at TEXT NOT NULL
);
`)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *Store) UpsertDownload(d *Download) (int64, error) {
	now := time.Now().UTC()
	if d.Status == "" {
		d.Status = "queued"
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now

	res, err := s.db.Exec(`
INSERT INTO downloads (
	item_id, item_name, item_type, series_id, season_number, episode_number,
	status, bytes_total, bytes_done, path, error, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(item_id, path) DO UPDATE SET
	item_name=excluded.item_name,
	item_type=excluded.item_type,
	series_id=excluded.series_id,
	season_number=excluded.season_number,
	episode_number=excluded.episode_number,
	status=excluded.status,
	bytes_total=excluded.bytes_total,
	bytes_done=excluded.bytes_done,
	error=excluded.error,
	updated_at=excluded.updated_at
`,
		d.ItemID,
		d.ItemName,
		d.ItemType,
		nullString(d.SeriesID),
		nullInt(d.SeasonNumber),
		nullInt(d.EpisodeNumber),
		d.Status,
		nullInt(d.BytesTotal),
		nullInt(d.BytesDone),
		d.Path,
		nullString(d.Error),
		d.CreatedAt.Format(time.RFC3339Nano),
		d.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("upsert download: %w", err)
	}

	id, _ := res.LastInsertId()
	if id == 0 {
		row := s.db.QueryRow("SELECT id FROM downloads WHERE item_id = ? AND path = ?", d.ItemID, d.Path)
		if err := row.Scan(&id); err != nil {
			return 0, fmt.Errorf("resolve download id: %w", err)
		}
	}
	return id, nil
}

func (s *Store) UpdateDownloadProgress(id int64, bytesDone, bytesTotal int64) error {
	_, err := s.db.Exec(`UPDATE downloads SET bytes_done = ?, bytes_total = ?, updated_at = ? WHERE id = ?`, bytesDone, bytesTotal, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("update download progress: %w", err)
	}
	return nil
}

func (s *Store) SetDownloadStatus(id int64, status string, errMsg string) error {
	_, err := s.db.Exec(`UPDATE downloads SET status = ?, error = ?, updated_at = ? WHERE id = ?`, status, nullString(errMsg), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("update download status: %w", err)
	}
	return nil
}

func (s *Store) ListDownloads(status string) ([]Download, error) {
	query := `SELECT id, item_id, item_name, item_type, series_id, season_number, episode_number, status, bytes_total, bytes_done, path, error, created_at, updated_at FROM downloads`
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list downloads: %w", err)
	}
	defer rows.Close()

	var out []Download
	for rows.Next() {
		var d Download
		var created, updated string
		if err := rows.Scan(&d.ID, &d.ItemID, &d.ItemName, &d.ItemType, &d.SeriesID, &d.SeasonNumber, &d.EpisodeNumber, &d.Status, &d.BytesTotal, &d.BytesDone, &d.Path, &d.Error, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan download: %w", err)
		}
		d.CreatedAt = parseTime(created)
		d.UpdatedAt = parseTime(updated)
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) GetDownload(id int64) (*Download, error) {
	row := s.db.QueryRow(`SELECT id, item_id, item_name, item_type, series_id, season_number, episode_number, status, bytes_total, bytes_done, path, error, created_at, updated_at FROM downloads WHERE id = ?`, id)
	var d Download
	var created, updated string
	if err := row.Scan(&d.ID, &d.ItemID, &d.ItemName, &d.ItemType, &d.SeriesID, &d.SeasonNumber, &d.EpisodeNumber, &d.Status, &d.BytesTotal, &d.BytesDone, &d.Path, &d.Error, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get download: %w", err)
	}
	d.CreatedAt = parseTime(created)
	d.UpdatedAt = parseTime(updated)
	return &d, nil
}

func (s *Store) UpdateSeriesProgress(seriesID string, season, episode int64) error {
	if seriesID == "" {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO series_progress (series_id, last_season, last_episode, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(series_id) DO UPDATE SET
	last_season=excluded.last_season,
	last_episode=excluded.last_episode,
	updated_at=excluded.updated_at
`, seriesID, season, episode, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("update series progress: %w", err)
	}
	return nil
}

func nullString(v interface{}) interface{} {
	switch x := v.(type) {
	case sql.NullString:
		if x.Valid {
			return x.String
		}
		return nil
	case string:
		if x == "" {
			return nil
		}
		return x
	default:
		return v
	}
}

func nullInt(v interface{}) interface{} {
	switch x := v.(type) {
	case sql.NullInt64:
		if x.Valid {
			return x.Int64
		}
		return nil
	case int64:
		return x
	case int:
		return int64(x)
	default:
		return v
	}
}

func parseTime(v string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}
	}
	return t
}
