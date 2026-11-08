//go:build !linux

package netns

import "errors"

func GetNetnsFd(netnsPath string) (int, error) {
	return -1, errors.New("GetNetnsFd is not applicable on non-linux")
}

func NetNsExecute(netnsPath string, operation func(netnsPath string) error) error {
	return errors.New("NetNsExecute is not applicable on non-linux")
}
