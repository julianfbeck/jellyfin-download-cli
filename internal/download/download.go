package download

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultChunkSize = 256 * 1024
)

var filenameCleaner = regexp.MustCompile(`[^a-zA-Z0-9._\- ]+`)

func SanitizeFileName(name string) string {
	clean := strings.TrimSpace(name)
	clean = filenameCleaner.ReplaceAllString(clean, "_")
	clean = strings.Trim(clean, "._ ")
	if clean == "" {
		return "download"
	}
	return clean
}

func CopyWithProgress(ctx context.Context, dst io.Writer, src io.Reader, total int64, limiter *rate.Limiter, onProgress func(int64, int64)) (int64, error) {
	buf := make([]byte, defaultChunkSize)
	var written int64
	lastUpdate := time.Now()

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if limiter != nil {
				if err := limiter.WaitN(ctx, n); err != nil {
					return written, err
				}
			}
			wn, writeErr := dst.Write(buf[:n])
			written += int64(wn)
			if writeErr != nil {
				return written, writeErr
			}
		}

		if time.Since(lastUpdate) > 750*time.Millisecond {
			lastUpdate = time.Now()
			if onProgress != nil {
				onProgress(written, total)
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				if onProgress != nil {
					onProgress(written, total)
				}
				return written, nil
			}
			return written, readErr
		}
	}
}

func ParseRateLimit(rateStr string) (*rate.Limiter, error) {
	rateStr = strings.TrimSpace(rateStr)
	if rateStr == "" {
		return nil, nil
	}

	value, unit, err := splitNumberUnit(rateStr)
	if err != nil {
		return nil, err
	}

	multiplier := float64(1)
	switch strings.ToUpper(unit) {
	case "B", "":
		multiplier = 1
	case "K", "KB", "KIB":
		multiplier = 1024
	case "M", "MB", "MIB":
		multiplier = 1024 * 1024
	case "G", "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	default:
		return nil, fmt.Errorf("unknown rate unit: %s", unit)
	}

	bytesPerSec := value * multiplier
	if bytesPerSec <= 0 {
		return nil, fmt.Errorf("rate must be > 0")
	}

	limit := rate.Limit(bytesPerSec)
	return rate.NewLimiter(limit, int(bytesPerSec)), nil
}

func splitNumberUnit(input string) (float64, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, "", fmt.Errorf("empty rate")
	}

	idx := 0
	for idx < len(input) {
		ch := input[idx]
		if (ch >= '0' && ch <= '9') || ch == '.' {
			idx++
			continue
		}
		break
	}

	numPart := strings.TrimSpace(input[:idx])
	unitPart := strings.TrimSpace(input[idx:])
	if numPart == "" {
		return 0, "", fmt.Errorf("missing rate value")
	}

	val, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid rate value: %w", err)
	}
	return val, unitPart, nil
}

func DefaultDownloadPath(baseDir, name, ext string) string {
	fileName := SanitizeFileName(name)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return filepath.Join(baseDir, fileName+ext)
}
