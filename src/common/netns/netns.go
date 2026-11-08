//go:build linux

package netns

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

func GetNetnsFd(netnsPath string) (int, error) {
	fd, err := syscall.Open(netnsPath, syscall.O_RDONLY, 0666)
	if err != nil {
		return -1, err
	}

	return fd, nil
}

func setns(netnsFd int) error {
	_, _, e1 := syscall.Syscall(unix.SYS_SETNS, uintptr(netnsFd), syscall.CLONE_NEWNET, 0)
	if e1 != 0 {
		return e1
	}

	return nil
}

func netNsExecute(netnsPath string, operation func(netnsPath string) error) (err error) {
	runtime.LockOSThread()

	origNetnsFd, err := GetNetnsFd("/proc/self/ns/net")
	if err != nil {
		runtime.UnlockOSThread()

		return err
	}

	defer syscall.Close(origNetnsFd)

	netnsFd, err := GetNetnsFd(netnsPath)
	if err != nil {
		runtime.UnlockOSThread()

		return err
	}

	defer syscall.Close(netnsFd)

	err = setns(netnsFd)
	if err != nil {
		runtime.UnlockOSThread()

		return err
	}

	defer func() {
		if setErr := setns(origNetnsFd); setErr != nil {
			err = errors.Join(err, fmt.Errorf("restore netns: %w", setErr))

			return
		}

		runtime.UnlockOSThread()
	}()

	if err := operation(netnsPath); err != nil {
		return err
	}

	return nil
}

func NetNsExecute(netnsPath string, operation func(netnsPath string) error) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- netNsExecute(netnsPath, operation)
	}()

	return <-errCh
}
