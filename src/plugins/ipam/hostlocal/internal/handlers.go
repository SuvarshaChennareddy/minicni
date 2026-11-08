//go:build linux

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/suvarshachennareddy/minicni/common/fileutils"
	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/ipam"
	"github.com/suvarshachennareddy/minicni/plugins/ipam/hostlocal/pkgs/ip"
	"github.com/suvarshachennareddy/minicni/plugins/skel"
)

func HandleAdd(cniArgs *skel.CmdArgs) error {
	conf, err := parseConfig(cniArgs.StdinData)
	if err != nil {
		return fmt.Errorf("could not parse config: %v", err)
	}

	_, subnetIpNet, err := net.ParseCIDR(conf.Subnet)
	if err != nil {
		return fmt.Errorf("could not parse subnet")
	}

	if subnetIpNet.IP.To4() == nil {
		return fmt.Errorf("ipv6 is not supported")
	}

	gatewayIp := net.ParseIP(conf.Gateway)
	if gatewayIp == nil {
		return fmt.Errorf("could not parse gateway IP")
	}

	if gatewayIp.To4() == nil {
		return fmt.Errorf("ipv6 is not supported")
	}

	ipamStateFilePath := getIpamStateFilePath(conf.Name)

	err = os.MkdirAll(filepath.Dir(ipamStateFilePath), 0755)
	if err != nil {
		return fmt.Errorf("could not create state file dir %s: %v", ipamStateFilePath, err)
	}

	stateFile, err := os.OpenFile(ipamStateFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not open/create state file %s: %v", ipamStateFilePath, err)
	}

	defer stateFile.Close()

	fileutils.LockFile(stateFile)
	defer fileutils.UnlockFile(stateFile)

	state, err := readStateFile(stateFile)
	if err != nil {
		return fmt.Errorf("could not read contents of state file %s: %v", ipamStateFilePath, err)
	}

	allocatedIP, err := findNextIP(state, subnetIpNet, gatewayIp)
	if err != nil {
		return fmt.Errorf("could not find IP to allocate: %v", err)
	}

	ipAddress := ip.FromNetIPtoIP(allocatedIP)
	state.IpAddresses[ipAddress] = types.AttachmentKey{
		ContainerID: cniArgs.CniContainerId,
		IfName:      cniArgs.CniIfname,
	}

	err = updateStateFile(stateFile, state)
	if err != nil {
		return fmt.Errorf("could not update state file %s: %v", ipamStateFilePath, err)
	}

	result := ipam.Result{
		Ips: []types.IP{
			{
				Address: (&net.IPNet{IP: allocatedIP, Mask: subnetIpNet.Mask}).String(),
				Gateway: conf.Gateway,
			}},
		Routes: conf.Routes,
	}

	err = json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		return fmt.Errorf("could not write result to stdout: %v", err)
	}

	return nil
}

func HandleDel(cniArgs *skel.CmdArgs) error {
	conf, err := parseConfig(cniArgs.StdinData)
	if err != nil {
		return fmt.Errorf("could not parse config: %v", err)
	}

	ipamStateFilePath := getIpamStateFilePath(conf.Name)

	stateFile, err := os.OpenFile(ipamStateFilePath, os.O_RDWR, 0666)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("could not open/create state file %s: %v", ipamStateFilePath, err)
	}

	defer stateFile.Close()

	fileutils.LockFile(stateFile)
	defer fileutils.UnlockFile(stateFile)

	attachmentKey := types.AttachmentKey{
		ContainerID: cniArgs.CniContainerId,
		IfName:      cniArgs.CniIfname,
	}

	err = deleteAllocatedIPInStateFile(stateFile, attachmentKey)
	if err != nil {
		return fmt.Errorf("could not deallocate ip in state file %s: %v", ipamStateFilePath, err)
	}

	return nil
}

func HandleCheck(cniArgs *skel.CmdArgs) error {
	conf, err := parseConfig(cniArgs.StdinData)
	if err != nil {
		return fmt.Errorf("could not parse config: %v", err)
	}

	if len(conf.PrevResult.Ips) == 0 {
		return fmt.Errorf("no ip addresses provided to check")
	}

	ipamStateFilePath := getIpamStateFilePath(conf.Name)

	stateFile, err := os.OpenFile(ipamStateFilePath, os.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("could not open state file %s: %v", ipamStateFilePath, err)
	}

	defer stateFile.Close()

	var allocatedIpStr string
	for i := range conf.PrevResult.Ips {
		if conf.PrevResult.Ips[i].Interface == nil || *conf.PrevResult.Ips[i].Interface >= len(conf.PrevResult.Interfaces) {
			return fmt.Errorf("no interface associated with the ip address %s", conf.PrevResult.Ips[i].Address)
		}

		if conf.PrevResult.Interfaces[*conf.PrevResult.Ips[i].Interface].Name == cniArgs.CniIfname {
			allocatedIpStr = conf.PrevResult.Ips[i].Address
		}
	}

	if allocatedIpStr == "" {
		return fmt.Errorf("could not find ip address for namespace %s's interface %s", cniArgs.CniNetNs, cniArgs.CniIfname)
	}

	parsedIp, _, err := net.ParseCIDR(allocatedIpStr)
	if err != nil {
		return fmt.Errorf("ip address of namespace %s's interface %s is not given or invalid", cniArgs.CniNetNs, cniArgs.CniIfname)
	}

	ipAddress := ip.FromNetIPtoIP(parsedIp)

	fileutils.LockFile(stateFile)
	defer fileutils.UnlockFile(stateFile)

	attachmentKey := types.AttachmentKey{
		ContainerID: cniArgs.CniContainerId,
		IfName:      cniArgs.CniIfname,
	}

	err = checkAllocatedIPInStateFile(stateFile, ipAddress, attachmentKey)
	if err != nil {
		return fmt.Errorf("ip check in state file %s failed: %v", ipamStateFilePath, err)
	}

	return nil
}
