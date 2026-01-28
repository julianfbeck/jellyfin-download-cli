package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/julianfbeck/jellyfin-download-cli/internal/api"
	"github.com/julianfbeck/jellyfin-download-cli/internal/download"
	"github.com/julianfbeck/jellyfin-download-cli/internal/store"
	"github.com/julianfbeck/jellyfin-download-cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

var (
	downloadRate   string
	downloadOutput string
	dryRun         bool
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download movies or episodes",
}

var downloadMovieCmd = &cobra.Command{
	Use:   "movie",
	Short: "Download a movie",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, cfg, storeDir, err := getClient(true)
		if err != nil {
			return err
		}

		var movieID string
		if movieSelect && !noInput {
			movieID, err = promptSelectMovie(client)
			if err != nil {
				return err
			}
		}
		if movieID == "" {
			movieID, err = resolveItemID(cmd, args, "Movie")
			if err != nil {
				return err
			}
		}

		item, err := client.GetItem(ctx, movieID)
		if err != nil {
			return exitError(4, err)
		}

		return runDownloadItems(client, storeDir, []api.Item{*item}, downloadOptions{
			Rate:    resolveRate(cfg.DefaultRate),
			Output:  downloadOutput,
			DryRun:  dryRun,
		})
	},
}

var (
	seriesID     string
	seasonList   string
	episodeList  string
	downloadAll  bool
	seriesSelect bool
	movieSelect  bool
)

var downloadSeriesCmd = &cobra.Command{
	Use:   "series",
	Short: "Download a series",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, cfg, storeDir, err := getClient(true)
		if err != nil {
			return err
		}

		id := seriesID
		if id == "" && seriesSelect && !noInput {
			id, err = promptSelectSeries(client)
			if err != nil {
				return err
			}
		}
		if id == "" {
			return exitError(2, fmt.Errorf("series id required (use --id or --select)"))
		}

		episodes, err := client.SeriesEpisodes(ctx, id)
		if err != nil {
			return exitError(4, err)
		}
		if len(episodes) == 0 {
			printInfo("No episodes found\n")
			return nil
		}

		seasons := parseNumberList(seasonList)
		episodesFilter := parseNumberList(episodeList)

		filtered := filterEpisodes(episodes, seasons, episodesFilter)
		if len(filtered) == 0 {
			printInfo("No episodes matched filters\n")
			return nil
		}

		if noInput && !downloadAll && seasonList == "" && episodeList == "" && !dryRun {
			return exitError(2, fmt.Errorf("non-interactive mode requires --all or --season/--episode filters"))
		}

		if !downloadAll && seasonList == "" && episodeList == "" && !dryRun {
			if !noInput {
				ok, err := confirmPrompt("Download all episodes? [y/N]: ")
				if err != nil {
					return err
				}
				if !ok {
					return exitError(2, fmt.Errorf("aborted"))
				}
			}
		}

		return runDownloadItems(client, storeDir, filtered, downloadOptions{
			Rate:    resolveRate(cfg.DefaultRate),
			Output:  downloadOutput,
			DryRun:  dryRun,
			Series:  id,
		})
	},
}

var downloadEpisodeCmd = &cobra.Command{
	Use:   "episode",
	Short: "Download a specific episode by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		itemID, err := resolveItemID(cmd, args, "Episode")
		if err != nil {
			return err
		}
		client, cfg, storeDir, err := getClient(true)
		if err != nil {
			return err
		}

		item, err := client.GetItem(ctx, itemID)
		if err != nil {
			return exitError(4, err)
		}

		return runDownloadItems(client, storeDir, []api.Item{*item}, downloadOptions{
			Rate:    resolveRate(cfg.DefaultRate),
			Output:  downloadOutput,
			DryRun:  dryRun,
		})
	},
}

func init() {
	downloadCmd.PersistentFlags().StringVar(&downloadRate, "rate", "", "Download rate limit (e.g. 5M, 500K)")
	downloadCmd.PersistentFlags().StringVar(&downloadOutput, "output", "", "Output directory (default: store/downloads)")
	downloadCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show planned downloads without downloading")

	downloadMovieCmd.Flags().String("id", "", "Movie item ID")
	downloadMovieCmd.Flags().BoolVar(&movieSelect, "select", false, "Interactively select a movie")
	_ = downloadMovieCmd.Flags().Lookup("id")

	downloadSeriesCmd.Flags().StringVar(&seriesID, "id", "", "Series item ID")
	downloadSeriesCmd.Flags().StringVar(&seasonList, "season", "", "Season numbers (e.g. 1,2,3-5)")
	downloadSeriesCmd.Flags().StringVar(&episodeList, "episode", "", "Episode numbers (e.g. 1,2,3-5)")
	downloadSeriesCmd.Flags().BoolVar(&downloadAll, "all", false, "Download all episodes")
	downloadSeriesCmd.Flags().BoolVar(&seriesSelect, "select", false, "Interactively select a series")

	downloadEpisodeCmd.Flags().String("id", "", "Episode item ID")
	_ = downloadEpisodeCmd.Flags().Lookup("id")

	downloadCmd.AddCommand(downloadMovieCmd)
	downloadCmd.AddCommand(downloadSeriesCmd)
	downloadCmd.AddCommand(downloadEpisodeCmd)
	rootCmd.AddCommand(downloadCmd)
}

type downloadOptions struct {
	Rate    string
	Output  string
	DryRun  bool
	Series  string
	OverridePath string
}

func resolveRate(defaultRate string) string {
	if downloadRate != "" {
		return downloadRate
	}
	return defaultRate
}

func resolveItemID(cmd *cobra.Command, args []string, itemType string) (string, error) {
	flag := cmd.Flags().Lookup("id")
	if flag != nil && flag.Value.String() != "" {
		return flag.Value.String(), nil
	}
	if len(args) > 0 {
		return args[0], nil
	}
	if noInput {
		return "", exitError(2, fmt.Errorf("%s id required", strings.ToLower(itemType)))
	}
	return promptLine(fmt.Sprintf("Enter %s ID: ", itemType), ""), nil
}

func runDownloadItems(client *api.Client, storeDir string, items []api.Item, opts downloadOptions) error {
	storeDB, err := store.Open(storeDir)
	if err != nil {
		return err
	}
	defer storeDB.Close()

	outputDir := opts.Output
	if outputDir == "" {
		outputDir = filepath.Join(storeDir, "downloads")
	}
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return err
	}

	limiter, err := download.ParseRateLimit(opts.Rate)
	if err != nil {
		return exitError(2, err)
	}

	for _, item := range items {
		if err := downloadItem(client, storeDB, item, outputDir, limiter, opts); err != nil {
			return err
		}
	}
	return nil
}

func downloadItem(client *api.Client, storeDB *store.Store, item api.Item, outputDir string, limiter *rate.Limiter, opts downloadOptions) error {
	path := opts.OverridePath
	if path == "" {
		name := buildItemFilename(item)
		ext := fileExtension(item.Path)
		path = download.DefaultDownloadPath(outputDir, name, ext)
	}

	record := &store.Download{
		ItemID:   item.Id,
		ItemName: item.Name,
		ItemType: item.Type,
		SeriesID: sqlNullString(opts.Series),
	}
	if item.ParentIndexNumber != 0 {
		record.SeasonNumber = sqlNullInt(item.ParentIndexNumber)
	}
	if item.IndexNumber != 0 {
		record.EpisodeNumber = sqlNullInt(item.IndexNumber)
	}
	record.Path = path

	id, err := storeDB.UpsertDownload(record)
	if err != nil {
		return err
	}

	if opts.DryRun {
		printInfo("[dry-run] %s -> %s\n", item.Name, path)
		return nil
	}

	if err := storeDB.SetDownloadStatus(id, "downloading", ""); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	offset := existingFileSize(path)
	if offset > 0 {
		printInfo("Resuming %s (%d bytes)\n", item.Name, offset)
	}

	resp, err := client.OpenDownload(ctx, item.Id, offset)
	if err != nil {
		_ = storeDB.SetDownloadStatus(id, "failed", err.Error())
		return exitError(5, err)
	}
	defer resp.Body.Close()

	if offset > 0 && resp.StatusCode == http.StatusOK {
		// Server did not honor range requests; restart download.
		offset = 0
	}

	if opts.OverridePath == "" {
		if filename := filenameFromResponse(resp); filename != "" {
			path = filepath.Join(filepath.Dir(path), download.SanitizeFileName(filename))
			record.Path = path
			_, _ = storeDB.UpsertDownload(record)
		}
	}

	f, err := openDownloadFile(path, offset)
	if err != nil {
		_ = storeDB.SetDownloadStatus(id, "failed", err.Error())
		return err
	}
	defer f.Close()

	bytesTotal := totalBytesFromResponse(resp, offset)
	_ = storeDB.UpdateDownloadProgress(id, offset, bytesTotal)

	lastPersist := time.Now()
	progressFn := func(written int64, total int64) {
		if time.Since(lastPersist) > 1*time.Second {
			_ = storeDB.UpdateDownloadProgress(id, offset+written, total)
			lastPersist = time.Now()
		}
		if !quietMode {
			printProgress(item.Name, offset+written, total)
		}
	}

	_, err = download.CopyWithProgress(ctx, f, resp.Body, bytesTotal, limiter, progressFn)
	if err != nil {
		_ = storeDB.SetDownloadStatus(id, "failed", err.Error())
		return exitError(5, err)
	}

	_ = storeDB.UpdateDownloadProgress(id, bytesTotal, bytesTotal)
	_ = storeDB.SetDownloadStatus(id, "done", "")
	if item.Type == "Episode" {
		_ = storeDB.UpdateSeriesProgress(opts.Series, int64(item.ParentIndexNumber), int64(item.IndexNumber))
	}

	if !quietMode {
		printInfo("Downloaded %s\n", item.Name)
	}
	return nil
}

func promptSelectSeries(client *api.Client) (string, error) {
	items, err := client.SearchItems(ctx, "", []string{"Series"}, 50)
	if err != nil {
		return "", exitError(4, err)
	}
	if len(items) == 0 {
		return "", exitError(2, fmt.Errorf("no series available"))
	}
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = formatItemLabel(item)
	}
	indices, err := ui.PromptSelectIndices("Select series:", labels, false)
	if err != nil {
		return "", err
	}
	if len(indices) == 0 {
		return "", exitError(2, fmt.Errorf("no series selected"))
	}
	return items[indices[0]].Id, nil
}

func promptSelectMovie(client *api.Client) (string, error) {
	items, err := client.SearchItems(ctx, "", []string{"Movie"}, 50)
	if err != nil {
		return "", exitError(4, err)
	}
	if len(items) == 0 {
		return "", exitError(2, fmt.Errorf("no movies available"))
	}
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = formatItemLabel(item)
	}
	indices, err := ui.PromptSelectIndices("Select movie:", labels, false)
	if err != nil {
		return "", err
	}
	if len(indices) == 0 {
		return "", exitError(2, fmt.Errorf("no movie selected"))
	}
	return items[indices[0]].Id, nil
}

func filterEpisodes(items []api.Item, seasons, episodes []int) []api.Item {
	seasonSet := toSet(seasons)
	episodeSet := toSet(episodes)

	var filtered []api.Item
	for _, item := range items {
		if item.Type != "Episode" && item.Type != "" {
			continue
		}
		if len(seasonSet) > 0 {
			if _, ok := seasonSet[item.ParentIndexNumber]; !ok {
				continue
			}
		}
		if len(episodeSet) > 0 {
			if _, ok := episodeSet[item.IndexNumber]; !ok {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ParentIndexNumber == filtered[j].ParentIndexNumber {
			return filtered[i].IndexNumber < filtered[j].IndexNumber
		}
		return filtered[i].ParentIndexNumber < filtered[j].ParentIndexNumber
	})
	return filtered
}

func parseNumberList(value string) []int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var out []int
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || start > end {
				continue
			}
			for i := start; i <= end; i++ {
				out = append(out, i)
			}
			continue
		}
		if v, err := strconv.Atoi(part); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func toSet(values []int) map[int]struct{} {
	set := make(map[int]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

func confirmPrompt(label string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func buildItemFilename(item api.Item) string {
	if item.Type == "Episode" {
		series := item.SeriesName
		if series == "" {
			series = "Series"
		}
		season := item.ParentIndexNumber
		ep := item.IndexNumber
		if season > 0 && ep > 0 {
			return fmt.Sprintf("%s - S%02dE%02d - %s", series, season, ep, item.Name)
		}
		return fmt.Sprintf("%s - %s", series, item.Name)
	}
	if item.ProductionYear > 0 {
		return fmt.Sprintf("%s (%d)", item.Name, item.ProductionYear)
	}
	return item.Name
}

func fileExtension(path string) string {
	if path == "" {
		return ".mkv"
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return ".mkv"
	}
	return ext
}

func existingFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func openDownloadFile(path string, offset int64) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(path, flags, 0600)
}

func filenameFromResponse(resp *http.Response) string {
	disposition := resp.Header.Get("Content-Disposition")
	if disposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func totalBytesFromResponse(resp *http.Response, offset int64) int64 {
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		parts := strings.Split(cr, "/")
		if len(parts) == 2 {
			if total, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
				return total
			}
		}
	}
	if resp.ContentLength > 0 {
		return resp.ContentLength + offset
	}
	return 0
}

func printProgress(name string, done, total int64) {
	if total > 0 {
		percent := float64(done) / float64(total) * 100
		fmt.Printf("\r%s: %.1f%% (%s/%s)", truncateName(name), percent, formatBytes(done), formatBytes(total))
	} else {
		fmt.Printf("\r%s: %s", truncateName(name), formatBytes(done))
	}
}

func truncateName(name string) string {
	if len(name) <= 40 {
		return name
	}
	return name[:37] + "..."
}

func formatBytes(v int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	if v >= GB {
		return fmt.Sprintf("%.2fGB", float64(v)/float64(GB))
	}
	if v >= MB {
		return fmt.Sprintf("%.2fMB", float64(v)/float64(MB))
	}
	if v >= KB {
		return fmt.Sprintf("%.2fKB", float64(v)/float64(KB))
	}
	return fmt.Sprintf("%dB", v)
}

func sqlNullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func sqlNullInt(value int) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}
