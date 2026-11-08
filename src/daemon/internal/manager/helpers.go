package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/suvarshachennareddy/minicni/common/types"
)

func getInode(path string) (uint64, error) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("could not stat file: path: %v", path)
	}

	return stat.Ino, nil
}

func isNetworkConfig(path, network string) *types.NetworkConfig {
	networkConfig := &types.NetworkConfig{}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("could not open path: %v", err)

		return nil
	}

	err = json.Unmarshal(content, networkConfig)
	if err != nil {
		log.Printf("error unmarshalling: %v", err)

		return nil
	}

	if networkConfig.Name != network {
		return nil
	}

	return networkConfig
}
