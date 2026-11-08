package ip

import (
	"encoding/binary"
	"net"
)

type IP uint32

func (ip IP) ToNetIp() net.IP {
	ipBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ipBytes, uint32(ip))

	return net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
}

func FromNetIPtoIP(netIp net.IP) IP {
	netIPAsBytes := []byte(netIp.To4())

	return IP(binary.BigEndian.Uint32(netIPAsBytes))
}
