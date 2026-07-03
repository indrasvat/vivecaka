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

// InboxCachePath returns the personal inbox cache path for a GitHub login.
func InboxCachePath(username string) string {
	name := sanitizeInboxPathComponent(strings.ToLower(strings.TrimSpace(username)))
	if name == "_" {
		name = "default"
	}
	return filepath.Join(config.CacheDir(), "inbox", name+".json")
}

func sanitizeInboxPathComponent(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" {
		s = "_"
	}
	return s
}

type inboxCacheFile struct {
	Version   int                `json:"version"`
	Username  string             `json:"username"`
	UpdatedAt time.Time          `json:"updated_at"`
	Result    domain.InboxResult `json:"result"`
}

// SaveInbox writes an attention inbox result atomically.
func SaveInbox(username string, result domain.InboxResult) error {
	path := InboxCachePath(username)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create inbox cache dir: %w", err)
	}

	now := time.Now()
	result.CachedAt = now
	for i := range result.Items {
		result.Items[i].Cached = true
	}
	data := inboxCacheFile{
		Version:   1,
		Username:  username,
		UpdatedAt: now,
		Result:    result,
	}
	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal inbox cache: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("write inbox cache: %w", err)
	}
	return os.Rename(tmp, path)
}

// LoadInbox reads the cached attention inbox. Missing cache is not an error.
func LoadInbox(username string) (domain.InboxResult, time.Time, error) {
	path := InboxCachePath(username)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.InboxResult{}, time.Time{}, nil
		}
		return domain.InboxResult{}, time.Time{}, fmt.Errorf("read inbox cache: %w", err)
	}

	var data inboxCacheFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.InboxResult{}, time.Time{}, fmt.Errorf("unmarshal inbox cache: %w", err)
	}
	if username != "" && !strings.EqualFold(data.Username, username) {
		return domain.InboxResult{}, time.Time{}, nil
	}
	data.Result.CachedAt = data.UpdatedAt
	for i := range data.Result.Items {
		data.Result.Items[i].Cached = true
	}
	return data.Result, data.UpdatedAt, nil
}
