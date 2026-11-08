package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/suvarshachennareddy/minicni/common/constants"
	"github.com/suvarshachennareddy/minicni/plugins/ipam"
)

// TODO: call by IPAM type
func callIpamPlugin(input []byte, ipamType string) ([]byte, error) {
	pluginPath := os.Getenv(constants.PluginBinPathEnv)

	cmd := exec.Command(pluginPath + "/" + ipamType)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("exit error %v: %s", err, exitErr.Stderr)
		}

		return nil, err
	}

	return output, nil
}

func allocateIP(input []byte, ipamType string) (*ipam.Result, error) {
	output, err := callIpamPlugin(input, ipamType)
	if err != nil {
		return nil, fmt.Errorf("could not successfully run ipam: %v", err)
	}

	ipamResult := &ipam.Result{}

	err = json.Unmarshal(output, ipamResult)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal ipam result: %v", err)
	}

	if len(ipamResult.Ips) == 0 {
		return nil, fmt.Errorf("ipam result does not contain IPs: %+v", ipamResult)
	}

	return ipamResult, nil
}

func deallocateIP(input []byte, ipamType string) error {
	_, err := callIpamPlugin(input, ipamType)
	if err != nil {
		return fmt.Errorf("could not successfully run ipam: %v", err)
	}

	return nil
}

func checkIpamState(input []byte, ipamType string) error {
	_, err := callIpamPlugin(input, ipamType)
	if err != nil {
		return fmt.Errorf("could not successfully run ipam: %v", err)
	}

	return nil
}
