//go:build linux

package fileutils

import (
	"os"

	"golang.org/x/sys/unix"
)

// LockFile blocks until the file is unlocked
func LockFile(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX)
	if err != nil {
		return err
	}

	return nil

}

func ShareLockFile(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_SH)
	if err != nil {
		return err
	}

	return nil

}

func UnlockFile(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_UN)
	if err != nil {
		return err
	}

	return nil
}
