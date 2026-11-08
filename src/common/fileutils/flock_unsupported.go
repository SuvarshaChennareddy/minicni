//go:build !linux

package fileutils

import (
	"errors"
	"os"
)

func LockFile(f *os.File) error {
	return errors.New("LockFile is not supported on non-linux")

}

func UnlockFile(f *os.File) error {
	return errors.New("UnlockFile is not supported on non-linux")
}
