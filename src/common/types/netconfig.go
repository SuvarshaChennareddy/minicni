package types

import "encoding/json"

type NetworkConfig struct {
	CniVersion  string            `json:"cniVersion"`
	CniVersions []string          `json:"cniVersions,omitempty"`
	Name        string            `json:"name"`
	Plugins     []json.RawMessage `json:"plugins"`
}
