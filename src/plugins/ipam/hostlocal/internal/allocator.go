//go:build linux

package internal

import (
	"fmt"
	"net"

	"github.com/suvarshachennareddy/minicni/plugins/ipam/hostlocal/pkgs/ip"
)

func findNextIP(state *ipamState, ipNet *net.IPNet, gateway net.IP) (net.IP, error) {
	lastIP := ip.FromNetIPtoIP(ipNet.IP.To4()) + 1
	ones, n := ipNet.Mask.Size()
	if n != 32 {
		return nil, fmt.Errorf("subnet mask size is not 32: %v", ipNet.String())
	}

	broadcast := ip.FromNetIPtoIP(ipNet.IP.To4()) | ((1 << (n - ones)) - 1)

	for {
		if lastIP == broadcast {
			break
		}

		if gateway.Equal(lastIP.ToNetIp()) {
			lastIP++
			continue
		}

		if _, found := state.IpAddresses[lastIP]; found {
			lastIP++
			continue
		}

		return lastIP.ToNetIp(), nil
	}

	return nil, fmt.Errorf("could not find an ip address in the subnet %v: %v", ipNet, lastIP.ToNetIp().String())
}
