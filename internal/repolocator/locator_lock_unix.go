//go:build !windows

package repolocator

import (
	"fmt"
	"math"
	"os"
	"syscall"
)

func lockFile(f *os.File) error {
	fd, err := flockFD(f)
	if err != nil {
		return err
	}
	return syscall.Flock(fd, syscall.LOCK_EX)
}

func unlockFile(f *os.File) {
	fd, err := flockFD(f)
	if err != nil {
		return
	}
	_ = syscall.Flock(fd, syscall.LOCK_UN)
}

func flockFD(f *os.File) (int, error) {
	fd := f.Fd()
	if fd > uintptr(math.MaxInt) {
		return 0, fmt.Errorf("file descriptor %d overflows int", fd)
	}
	return int(fd), nil
}
