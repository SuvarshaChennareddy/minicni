package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"syscall"

	"github.com/suvarshachennareddy/minicni/common/netns"
	"github.com/vishvananda/netlink"
)

func makeBridge(name string) (netlink.Link, error) {
	bridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: name}}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return nil, err
	}

	return bridge, nil
}

// TODO: should we assign mac addresses?
func makeVethPair(vethName, vethPeerName string) (veth, vethPeer netlink.Link, err error) {
	vethTmp := netlink.NewVeth(netlink.LinkAttrs{Name: vethName})
	vethTmp.PeerName = vethPeerName

	if err := netlink.LinkAdd(vethTmp); err != nil {
		return nil, nil, err
	}

	veth, err = netlink.LinkByName(vethTmp.Attrs().Name)
	if err != nil {
		return nil, nil, err
	}

	vethPeer, err = netlink.LinkByName(vethTmp.PeerName)
	if err != nil {
		return nil, nil, err
	}

	return veth, vethPeer, nil
}

func addLinkToNetns(veth netlink.Link, netnsPath string) error {
	fd, err := netns.GetNetnsFd(netnsPath)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	return netlink.LinkSetNsFd(veth, fd)
}

func generateVethPairNames(containerID, ifname string) (string, string) {
	sum := sha256.Sum256([]byte(containerID + "-" + ifname))
	veth := "veth" + hex.EncodeToString(sum[:])[:8]

	return veth, veth + "p"
}
