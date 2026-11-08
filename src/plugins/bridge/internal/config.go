package internal

import (
	"encoding/json"
	"fmt"

	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/ipam"
)

type Config struct {
	types.PluginConfigBase
	Bridge string `json:"bridge"`

	// TODO: implement isDefaultGateway and forceAddress
	IsGateway bool `json:"isGateway"`

	Ipam ipam.BaseConfig `json:"ipam"`

	PrevResult types.AttachmentBody `json:"prevResult"`
}

func parseConfig(stdinData []byte) (*Config, error) {
	conf := &Config{}

	err := json.Unmarshal(stdinData, conf)
	if err != nil {
		return nil, err
	}

	if conf.Type != "bridge" {
		return nil, fmt.Errorf("plugin type is not bridge, input given: %v", conf.Type)
	}

	return conf, nil
}
