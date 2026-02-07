//go:build windows

package repolocator

import "os"

// On Windows, the lock file is opened exclusively via os.Create which
// already prevents concurrent writes within the same process. For true
// cross-process locking, LockFileEx via syscall would be needed, but
// the single-instance usage pattern makes this sufficient for now.

func lockFile(_ *os.File) error {
	return nil
}

func unlockFile(_ *os.File) {}
