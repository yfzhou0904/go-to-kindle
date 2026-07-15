package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yfzhou0904/go-to-kindle/util"
)

// cleanupThrottle is the minimum interval between archive cleanup sweeps.
const cleanupThrottle = 24 * time.Hour

// maybeCleanupArchive removes archive files older than the configured
// retention period. It is best-effort and throttled to run at most once per
// day, so it is safe to call on every startup. A retention of 0 disables
// cleanup entirely.
func maybeCleanupArchive(retentionDays int) {
	if retentionDays <= 0 {
		return
	}

	marker := filepath.Join(util.BaseDir(), "last_cleanup")
	if info, err := os.Stat(marker); err == nil {
		if time.Since(info.ModTime()) < cleanupThrottle {
			return
		}
	}

	removed, err := cleanupArchive(filepath.Join(util.BaseDir(), "archive"), retentionDays, time.Now())
	if err != nil {
		// Leave the marker untouched so the sweep is retried on the next
		// startup rather than suppressed for a full day.
		log.Printf("archive cleanup: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("archive cleanup: removed %d file(s) older than %d day(s)", removed, retentionDays)
	}

	touchMarker(marker)
}

// cleanupArchive deletes regular files directly under dir whose modification
// time is older than retentionDays before now. It returns the number of files
// removed. A missing directory is not an error.
func cleanupArchive(dir string, retentionDays int, now time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// touchMarker records the time of the most recent cleanup sweep.
func touchMarker(path string) {
	now := time.Now()
	if err := os.Chtimes(path, now, now); err == nil {
		return
	}
	if f, err := os.Create(path); err == nil {
		f.Close()
	}
}
