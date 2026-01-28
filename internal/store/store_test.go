package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestStoreDownloadLifecycle(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	rec := &Download{
		ItemID:   "item-1",
		ItemName: "Test Movie",
		ItemType: "Movie",
		Path:     filepath.Join(dir, "downloads", "test.mkv"),
		Status:   "queued",
	}

	id, err := st.UpsertDownload(rec)
	if err != nil {
		t.Fatalf("UpsertDownload: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected id")
	}

	if err := st.UpdateDownloadProgress(id, 100, 1000); err != nil {
		t.Fatalf("UpdateDownloadProgress: %v", err)
	}
	if err := st.SetDownloadStatus(id, "downloading", ""); err != nil {
		t.Fatalf("SetDownloadStatus: %v", err)
	}

	row, err := st.GetDownload(id)
	if err != nil {
		t.Fatalf("GetDownload: %v", err)
	}
	if row == nil || row.Status != "downloading" {
		t.Fatalf("expected status=downloading, got %+v", row)
	}
	if !row.BytesDone.Valid || row.BytesDone.Int64 != 100 {
		t.Fatalf("expected bytes_done=100, got %+v", row.BytesDone)
	}

	rec.Status = "done"
	if _, err := st.UpsertDownload(rec); err != nil {
		t.Fatalf("UpsertDownload update: %v", err)
	}

	list, err := st.ListDownloads("")
	if err != nil {
		t.Fatalf("ListDownloads: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 download, got %d", len(list))
	}
	if list[0].Status != "done" {
		t.Fatalf("expected status=done, got %s", list[0].Status)
	}
}

func TestSeriesProgress(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	if err := st.UpdateSeriesProgress("series-1", 2, 5); err != nil {
		t.Fatalf("UpdateSeriesProgress: %v", err)
	}

	db, err := sql.Open("sqlite3", "file:"+DBPath(dir))
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	defer db.Close()

	var season, episode int64
	row := db.QueryRow("SELECT last_season, last_episode FROM series_progress WHERE series_id = ?", "series-1")
	if err := row.Scan(&season, &episode); err != nil {
		t.Fatalf("scan series_progress: %v", err)
	}
	if season != 2 || episode != 5 {
		t.Fatalf("unexpected progress: %d/%d", season, episode)
	}
}

func TestDBPath(t *testing.T) {
	got := DBPath("/tmp/jellyfin-test")
	want := filepath.Join("/tmp/jellyfin-test", "jellyfin.db")
	if got != want {
		t.Fatalf("DBPath unexpected: %s", got)
	}
}
