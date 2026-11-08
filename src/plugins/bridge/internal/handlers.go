package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/suvarshachennareddy/minicni/common/netns"
	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/skel"
	"github.com/vishvananda/netlink"
)

func HandleAdd(cniArgs *skel.CmdArgs) error {
	conf, err := parseConfig(cniArgs.StdinData)
	if err != nil {
		return fmt.Errorf("could not parse config: %v", err)
	}

	bridge, err := netlink.LinkByName(conf.Bridge)
	if errors.As(err, &netlink.LinkNotFoundError{}) {
		bridge, err = makeBridge(conf.Bridge)
		if err != nil {
			return fmt.Errorf("could not create bridge: %v", err)
		}

		err = netlink.LinkSetUp(bridge)
		if err != nil {
			return fmt.Errorf("could not bring bridge up: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("could not get bridge info: %v", err)
	}

	hostVethName, containerVethName := generateVethPairNames(cniArgs.CniContainerId, cniArgs.CniIfname)
	vethHost, vethContainer, err := makeVethPair(hostVethName, containerVethName)
	if err != nil {
		return fmt.Errorf("could not create veth pair: %v", err)
	}

	defer func() {
		if err != nil {
			if delErr := netlink.LinkDel(vethHost); delErr != nil {
				err = errors.Join(err, fmt.Errorf("could not delete veth pair: %v", delErr))
			}
		}
	}()

	err = netlink.LinkSetMaster(vethHost, bridge)
	if err != nil {
		return fmt.Errorf("could not attach %s to bridge: %v", hostVethName, err)
	}

	err = addLinkToNetns(vethContainer, cniArgs.CniNetNs)
	if err != nil {
		return fmt.Errorf("could not move %s to netns %s: %v", containerVethName, cniArgs.CniNetNs, err)
	}

	err = netlink.LinkSetUp(vethHost)
	if err != nil {
		return fmt.Errorf("could not bring %s up: %v", hostVethName, err)
	}

	ipamResult, err := allocateIP(cniArgs.StdinData, conf.Ipam.Type)
	if err != nil {
		return fmt.Errorf("could not allocate IP: %v", err)
	}

	allocatedIP, allocatedIPNet, err := net.ParseCIDR(ipamResult.Ips[0].Address)
	if err != nil {
		return fmt.Errorf("could not parse allocated IP: %v", err)
	}

	allocatedIPNet.IP = allocatedIP

	err = netns.NetNsExecute(cniArgs.CniNetNs, func(_ string) error {
		var innerErr error

		vethContainer, innerErr = netlink.LinkByName(containerVethName)
		if innerErr != nil {
			return innerErr
		}

		innerErr = netlink.LinkSetName(vethContainer, cniArgs.CniIfname)
		if innerErr != nil {
			return innerErr
		}

		innerErr = netlink.AddrAdd(vethContainer, &netlink.Addr{
			IPNet: allocatedIPNet,
		})
		if innerErr != nil {
			return fmt.Errorf("could not allocate IP: %v", innerErr)
		}

		vethContainer, innerErr = netlink.LinkByName(cniArgs.CniIfname)
		if innerErr != nil {
			return fmt.Errorf("could not refetch container end of veth: %v", innerErr)
		}

		return netlink.LinkSetUp(vethContainer)
	})
	if err != nil {
		return fmt.Errorf("could not configure %s in netns %s: %v", cniArgs.CniIfname, cniArgs.CniNetNs, err)
	}

	bridge, err = netlink.LinkByName(bridge.Attrs().Name)
	if err != nil {
		return fmt.Errorf("could not refetch bridge info: %v", err)
	}

	result := types.AttachmentBody{
		Ips:    ipamResult.Ips,
		Routes: ipamResult.Routes,
	}
	result.Ips[0].Interface = new(2)
	result.Interfaces = []types.Interface{
		{
			Name: vethHost.Attrs().Name,
			Mac:  vethHost.Attrs().HardwareAddr.String(),
		}, {
			Name: bridge.Attrs().Name,
			Mac:  bridge.Attrs().HardwareAddr.String(),
		}, {
			Name:    vethContainer.Attrs().Name,
			Mac:     vethContainer.Attrs().HardwareAddr.String(),
			Sandbox: cniArgs.CniNetNs,
		},
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

	hostVethName, _ := generateVethPairNames(cniArgs.CniContainerId, cniArgs.CniIfname)

	link, err := netlink.LinkByName(hostVethName)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}

		return fmt.Errorf("could not find bridge end of veth: %v", err)
	}

	err = netlink.LinkDel(link)
	if err != nil {
		return fmt.Errorf("could not delete veth: %v", err)
	}

	err = deallocateIP(cniArgs.StdinData, conf.Ipam.Type)
	if err != nil {
		return fmt.Errorf("could not deallocate IP: %v", err)
	}

	return nil
}

func HandleCheck(cniArgs *skel.CmdArgs) error {
	conf, err := parseConfig(cniArgs.StdinData)
	if err != nil {
		return fmt.Errorf("could not parse config: %v", err)
	}

	interfaceIndexToIpAddress := make(map[int]net.IP)
	for i := range conf.PrevResult.Ips {
		if conf.PrevResult.Ips[i].Interface == nil || *conf.PrevResult.Ips[i].Interface >= len(conf.PrevResult.Interfaces) {
			return fmt.Errorf("no interface associated with the ip address %s", conf.PrevResult.Ips[i].Address)
		}

		parsedIp, _, err := net.ParseCIDR(conf.PrevResult.Ips[i].Address)
		if err != nil {
			return fmt.Errorf("ip address ['%s'] is not given or invalid", conf.PrevResult.Ips[i].Address)
		}

		interfaceIndexToIpAddress[*conf.PrevResult.Ips[i].Interface] = parsedIp
	}

	bridge, err := netlink.LinkByName(conf.Bridge)
	if err != nil {
		return fmt.Errorf("could not fetch information of bridge: %v", err)
	}

	var vethContainer netlink.Link
	var vethContainerIpAddresses []netlink.Addr
	err = netns.NetNsExecute(cniArgs.CniNetNs, func(_ string) error {
		var innerErr error

		vethContainer, innerErr = netlink.LinkByName(cniArgs.CniIfname)
		if innerErr != nil {
			return innerErr
		}

		vethContainerIpAddresses, innerErr = netlink.AddrList(vethContainer, netlink.FAMILY_V4)
		if innerErr != nil {
			return fmt.Errorf("could not fetch IPs of container end of veth: %v", innerErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not fetch information of container end of veth: %v", err)
	}

	for i := range conf.PrevResult.Interfaces {
		switch conf.PrevResult.Interfaces[i].Name {
		case conf.Bridge:
			err = checkBridge(bridge)
		case cniArgs.CniIfname:
			err = checkContainerVeth(vethContainer, vethContainerIpAddresses, interfaceIndexToIpAddress[i], conf.PrevResult.Interfaces[i].Mac)
		default:
			err = checkHostVeth(bridge, vethContainer, conf.PrevResult.Interfaces[i].Name, conf.PrevResult.Interfaces[i].Mac)
		}

		if err != nil {
			return err
		}
	}

	err = checkIpamState(cniArgs.StdinData, conf.Ipam.Type)
	if err != nil {
		return fmt.Errorf("IPAM check failed: %v", err)
	}

	return nil
}

func checkBridge(bridge netlink.Link) error {
	if bridge.Type() != "bridge" {
		return fmt.Errorf("%s is not a bridge", bridge.Attrs().Name)
	}

	// we do not check mac address because it could change

	return nil
}

func checkHostVeth(bridge netlink.Link, vethContainer netlink.Link, interfaceName, macAddress string) error {
	vethHost, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return err
	}

	if vethHost.Type() != "veth" {
		return fmt.Errorf("%s is not a veth", interfaceName)
	}

	if vethHost.Attrs().HardwareAddr.String() != macAddress {
		return fmt.Errorf("recorded mac address %s for %s is incorrect", macAddress, interfaceName)
	}

	if vethHost.Attrs().MasterIndex != bridge.Attrs().Index {
		return fmt.Errorf("host end of veth is not attached to bridge")
	}

	peerIndex, err := netlink.VethPeerIndex(vethHost.(*netlink.Veth))
	if err != nil {
		return fmt.Errorf("could not get peer index of %s: %v", interfaceName, err)
	}

	if peerIndex != vethContainer.Attrs().Index {
		return fmt.Errorf("peer of %s has index %d, expected container interface %s with index %d",
			interfaceName, peerIndex, vethContainer.Attrs().Name, vethContainer.Attrs().Index)
	}

	return nil
}

func checkContainerVeth(vethContainer netlink.Link, ipAddresses []netlink.Addr, givenIpAddress net.IP, macAddress string) error {
	if vethContainer.Type() != "veth" {
		return fmt.Errorf("%s is not a veth", vethContainer.Attrs().Name)
	}

	if vethContainer.Attrs().HardwareAddr.String() != macAddress {
		return fmt.Errorf("recorded mac address %s for %s is incorrect", macAddress, vethContainer.Attrs().Name)
	}

	found := false
	for i := range ipAddresses {
		if ipAddresses[i].IP.Equal(givenIpAddress) {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("ip address %s is not associated with interface %s", givenIpAddress.String(), vethContainer.Attrs().Name)
	}

	return nil
}
