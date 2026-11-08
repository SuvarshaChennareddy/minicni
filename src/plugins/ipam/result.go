package ipam

import "github.com/suvarshachennareddy/minicni/common/types"

type Result struct {
	Ips    []types.IP    `json:"ips"`
	Routes []types.Route `json:"routes"`
}
