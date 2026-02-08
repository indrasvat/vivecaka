package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// CachePath returns the cache file path for a given repo.
func CachePath(repo domain.RepoRef) string {
	name := fmt.Sprintf("%s_%s.json", repo.Owner, repo.Name)
	return filepath.Join(config.CacheDir(), "repos", name)
}

// cacheFile is the JSON structure stored on disk.
type cacheFile struct {
	Version   int         `json:"version"`
	Repo      string      `json:"repo"`
	UpdatedAt time.Time   `json:"updated_at"`
	PRs       []domain.PR `json:"prs"`
}

// Save writes PR data to the cache file atomically.
func Save(repo domain.RepoRef, prs []domain.PR) error {
	path := CachePath(repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data := cacheFile{
		Version:   1,
		Repo:      repo.String(),
		UpdatedAt: time.Now(),
		PRs:       prs,
	}

	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	// Write to temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return os.Rename(tmp, path)
}

// Load reads cached PR data from disk.
// Returns nil, zero time, nil if no cache exists.
func Load(repo domain.RepoRef) ([]domain.PR, time.Time, error) {
	path := CachePath(repo)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, fmt.Errorf("read cache: %w", err)
	}

	var data cacheFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, time.Time{}, fmt.Errorf("unmarshal cache: %w", err)
	}

	// Verify repo matches.
	if !strings.EqualFold(data.Repo, repo.String()) {
		return nil, time.Time{}, nil
	}

	return data.PRs, data.UpdatedAt, nil
}

// IsStale returns true if the cache is older than the given TTL.
// Returns true if no cache exists.
func IsStale(repo domain.RepoRef, ttl time.Duration) bool {
	_, updated, err := Load(repo)
	if err != nil || updated.IsZero() {
		return true
	}
	return time.Since(updated) > ttl
}
