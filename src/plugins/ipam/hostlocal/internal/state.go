//go:build linux

package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/ipam/hostlocal/pkgs/ip"
)

type ipamState struct {
	IpAddresses map[ip.IP]types.AttachmentKey `json:"ipAddresses"`
}

func getIpamStateFilePath(netName string) string {
	return filepath.Join(os.TempDir(), netName, "ipamstate.json")
}

func readStateFile(stateFile *os.File) (*ipamState, error) {
	ipamState := &ipamState{
		IpAddresses: make(map[ip.IP]types.AttachmentKey),
	}

	content, err := io.ReadAll(stateFile)
	if err != nil {
		return nil, fmt.Errorf("could not read state file: %v", err)
	}

	if len(content) > 0 {
		err = json.Unmarshal(content, ipamState)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal state file contents: %v", err)
		}
	}

	return ipamState, nil
}

func updateStateFile(
	stateFile *os.File,
	state *ipamState,
) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not marshal state: %v", err)
	}

	err = stateFile.Truncate(0)
	if err != nil {
		return fmt.Errorf("could not truncate state file: %v", err)
	}

	_, err = stateFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("could not seek state file: %v", err)
	}

	_, err = stateFile.Write(bytes)
	if err != nil {
		return fmt.Errorf("could not update state: %v", err)
	}

	return nil
}

func deleteAllocatedIPInStateFile(stateFile *os.File, interfaceKeyToDel types.AttachmentKey) error {
	ipamState, err := readStateFile(stateFile)
	if err != nil {
		return fmt.Errorf("could not read state file: %v", err)
	}

	for ip, currInterfaceKey := range ipamState.IpAddresses {
		if interfaceKeyToDel == currInterfaceKey {
			delete(ipamState.IpAddresses, ip)
		}
	}

	err = updateStateFile(stateFile, ipamState)
	if err != nil {
		return fmt.Errorf("could not update state file: %v", err)
	}

	return nil
}

func checkAllocatedIPInStateFile(stateFile *os.File, ip ip.IP, expectedAttachementKey types.AttachmentKey) error {
	ipamState, err := readStateFile(stateFile)
	if err != nil {
		return fmt.Errorf("could not read state file: %v", err)
	}

	if ipamState.IpAddresses[ip] != expectedAttachementKey {
		return fmt.Errorf("ip in ipam state is not associated with the expected interface")
	}

	return nil
}
