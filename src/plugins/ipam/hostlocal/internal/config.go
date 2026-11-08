//go:build linux

package internal

import (
	"encoding/json"
	"fmt"

	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/ipam"
)

type IpamConfig struct {
	ipam.BaseConfig
	Subnet  string        `json:"subnet"`
	Gateway string        `json:"gateway"`
	Routes  []types.Route `json:"routes"`
}

type Config struct {
	Name       string `json:"name"`
	IpamConfig `json:"ipam"`
	PrevResult types.AttachmentBody `json:"prevResult"`
}

func parseConfig(stdinData []byte) (*Config, error) {
	conf := &Config{}

	err := json.Unmarshal(stdinData, conf)
	if err != nil {
		return nil, err
	}

	if conf.Type != "host-local" {
		return nil, fmt.Errorf("ipam type is not host-local, input given: %v", conf.Type)
	}

	return conf, nil
}
