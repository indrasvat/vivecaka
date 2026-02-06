package repolocator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// Locator manages the known-repos registry, mapping repositories to local paths.
// It auto-learns repo locations from app launches and managed clones.
type Locator struct {
	dataPath string // path to known-repos.json
}

// New creates a Locator using the XDG data directory.
func New() *Locator {
	return &Locator{
		dataPath: filepath.Join(config.DataDir(), "known-repos.json"),
	}
}

// NewWithPath creates a Locator with a custom data path (for testing).
func NewWithPath(dataPath string) *Locator {
	return &Locator{dataPath: dataPath}
}

// Lookup finds a known path for a repo. Returns ("", false) if not found.
func (l *Locator) Lookup(repo domain.RepoRef) (string, bool) {
	entries, err := l.load()
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if repoEqual(e.Repo, repo) {
			return e.Path, true
		}
	}
	return "", false
}

// Validate looks up a repo and verifies the path still exists and has the
// correct git remote. Returns the path and true if valid.
func (l *Locator) Validate(repo domain.RepoRef) (string, bool) {
	path, found := l.Lookup(repo)
	if !found {
		return "", false
	}
	if !isValidRepoDir(path, repo) {
		// Stale entry — remove it.
		_ = l.Remove(repo)
		return "", false
	}
	return path, true
}

// Register adds or updates a repo→path mapping with timestamp and source tag.
func (l *Locator) Register(repo domain.RepoRef, path, source string) error {
	return l.withLock(func() error {
		entries, err := l.load()
		if err != nil {
			entries = nil // start fresh on corrupt file
		}

		now := time.Now()
		found := false
		for i, e := range entries {
			if repoEqual(e.Repo, repo) {
				entries[i].Path = path
				entries[i].LastSeen = now
				entries[i].Source = source
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, domain.RepoLocation{
				Repo:     repo,
				Path:     path,
				LastSeen: now,
				Source:   source,
			})
		}
		return l.save(entries)
	})
}

// Remove deletes a repo entry from the registry.
func (l *Locator) Remove(repo domain.RepoRef) error {
	return l.withLock(func() error {
		entries, err := l.load()
		if err != nil {
			return nil // nothing to remove from
		}
		filtered := entries[:0]
		for _, e := range entries {
			if !repoEqual(e.Repo, repo) {
				filtered = append(filtered, e)
			}
		}
		return l.save(filtered)
	})
}

// All returns all known repo locations.
func (l *Locator) All() ([]domain.RepoLocation, error) {
	return l.load()
}

// CacheClonePath returns the deterministic managed clone path for a repo.
func (l *Locator) CacheClonePath(repo domain.RepoRef) string {
	return filepath.Join(config.CacheDir(), "clones", repo.Owner, repo.Name)
}

// withLock acquires an exclusive file lock around read-modify-write operations.
// This prevents concurrent vivecaka instances from losing each other's writes.
func (l *Locator) withLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(l.dataPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	lockPath := l.dataPath + ".lock"
	f, err := os.Create(lockPath)
	if err != nil {
		return fmt.Errorf("create lock file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

func (l *Locator) load() ([]domain.RepoLocation, error) {
	raw, err := os.ReadFile(l.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read known-repos: %w", err)
	}
	var entries []domain.RepoLocation
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal known-repos: %w", err)
	}
	return entries, nil
}

func (l *Locator) save(entries []domain.RepoLocation) error {
	if err := os.MkdirAll(filepath.Dir(l.dataPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal known-repos: %w", err)
	}
	// Atomic write: temp file + rename.
	tmp := l.dataPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write known-repos: %w", err)
	}
	return os.Rename(tmp, l.dataPath)
}

// repoEqual compares two RepoRefs case-insensitively (GitHub is case-insensitive).
func repoEqual(a, b domain.RepoRef) bool {
	return strings.EqualFold(a.Owner, b.Owner) && strings.EqualFold(a.Name, b.Name)
}

// isValidRepoDir checks if a path exists and contains a git repo matching the expected remote.
func isValidRepoDir(path string, expected domain.RepoRef) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil || !info.IsDir() {
		return false
	}
	// Check git remote matches.
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	remote := strings.TrimSpace(string(out))
	// Match either SSH or HTTPS remote format.
	expectedStr := strings.ToLower(expected.Owner + "/" + expected.Name)
	return strings.Contains(strings.ToLower(remote), expectedStr)
}
