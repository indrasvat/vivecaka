package repolocator

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func testLocator(t *testing.T) *Locator {
	t.Helper()
	dir := t.TempDir()
	return NewWithPath(filepath.Join(dir, "known-repos.json"))
}

func TestLookupMiss(t *testing.T) {
	loc := testLocator(t)
	path, found := loc.Lookup(domain.RepoRef{Owner: "foo", Name: "bar"})
	assert.False(t, found)
	assert.Empty(t, path)
}

func TestRegisterAndLookup(t *testing.T) {
	loc := testLocator(t)
	repo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}

	err := loc.Register(repo, "/Users/test/code/steipete/CodexBar", "detected")
	require.NoError(t, err)

	path, found := loc.Lookup(repo)
	assert.True(t, found)
	assert.Equal(t, "/Users/test/code/steipete/CodexBar", path)
}

func TestRegisterUpdatesExisting(t *testing.T) {
	loc := testLocator(t)
	repo := domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"}

	err := loc.Register(repo, "/old/path", "detected")
	require.NoError(t, err)

	err = loc.Register(repo, "/new/path", "manual")
	require.NoError(t, err)

	path, found := loc.Lookup(repo)
	assert.True(t, found)
	assert.Equal(t, "/new/path", path)

	// Should still be only one entry.
	entries, err := loc.All()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "manual", entries[0].Source)
}

func TestRemove(t *testing.T) {
	loc := testLocator(t)
	repo := domain.RepoRef{Owner: "foo", Name: "bar"}

	err := loc.Register(repo, "/some/path", "detected")
	require.NoError(t, err)

	err = loc.Remove(repo)
	require.NoError(t, err)

	_, found := loc.Lookup(repo)
	assert.False(t, found)
}

func TestRemoveNonexistent(t *testing.T) {
	loc := testLocator(t)
	err := loc.Remove(domain.RepoRef{Owner: "no", Name: "such"})
	assert.NoError(t, err)
}

func TestAll(t *testing.T) {
	loc := testLocator(t)

	err := loc.Register(domain.RepoRef{Owner: "a", Name: "1"}, "/a/1", "detected")
	require.NoError(t, err)
	err = loc.Register(domain.RepoRef{Owner: "b", Name: "2"}, "/b/2", "cloned")
	require.NoError(t, err)

	entries, err := loc.All()
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestCaseInsensitiveLookup(t *testing.T) {
	loc := testLocator(t)
	err := loc.Register(domain.RepoRef{Owner: "Steipete", Name: "CodexBar"}, "/path", "detected")
	require.NoError(t, err)

	path, found := loc.Lookup(domain.RepoRef{Owner: "steipete", Name: "codexbar"})
	assert.True(t, found)
	assert.Equal(t, "/path", path)
}

func TestCacheClonePath(t *testing.T) {
	loc := testLocator(t)
	repo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	p := loc.CacheClonePath(repo)
	assert.Contains(t, p, "clones")
	assert.Contains(t, p, "steipete")
	assert.Contains(t, p, "CodexBar")
}

func TestValidateWithNonexistentPath(t *testing.T) {
	loc := testLocator(t)
	repo := domain.RepoRef{Owner: "foo", Name: "bar"}

	err := loc.Register(repo, "/nonexistent/path/that/doesnt/exist", "detected")
	require.NoError(t, err)

	path, valid := loc.Validate(repo)
	assert.False(t, valid)
	assert.Empty(t, path)

	// Stale entry should be removed.
	_, found := loc.Lookup(repo)
	assert.False(t, found)
}

func TestCorruptedJSON(t *testing.T) {
	loc := testLocator(t)
	err := os.MkdirAll(filepath.Dir(loc.dataPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(loc.dataPath, []byte("not valid json{{{"), 0o644)
	require.NoError(t, err)

	// Lookup should gracefully return miss on corrupt data.
	_, found := loc.Lookup(domain.RepoRef{Owner: "a", Name: "b"})
	assert.False(t, found)

	// Register should recover from corrupt data.
	err = loc.Register(domain.RepoRef{Owner: "a", Name: "b"}, "/path", "detected")
	assert.NoError(t, err)

	path, found := loc.Lookup(domain.RepoRef{Owner: "a", Name: "b"})
	assert.True(t, found)
	assert.Equal(t, "/path", path)
}

func TestConcurrentRegister(t *testing.T) {
	loc := testLocator(t)
	var wg sync.WaitGroup

	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			repo := domain.RepoRef{Owner: "owner", Name: "repo"}
			_ = loc.Register(repo, "/path", "detected")
			_, _ = loc.Lookup(repo)
			_ = loc.Remove(domain.RepoRef{Owner: "nonexistent", Name: "repo"})
		}(i)
	}
	wg.Wait()
}

func TestEmptyFile(t *testing.T) {
	loc := testLocator(t)
	entries, err := loc.All()
	assert.NoError(t, err)
	assert.Nil(t, entries)
}
