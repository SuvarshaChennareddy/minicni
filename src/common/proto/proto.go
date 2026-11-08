package proto

import (
	"encoding/json"

	"github.com/suvarshachennareddy/minicni/common/types"
)

type CNIArgs struct {
	CniCommand     types.CNICommand
	NetworkName    string
	NetNsPath      string
	InterfaceName  string                     `json:",omitempty"`
	CapabilityArgs map[string]json.RawMessage `json:",omitempty"`
}

type StatusResponse struct {
	Success bool
	Error   string `json:",omitempty"`
}
