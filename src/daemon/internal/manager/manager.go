package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/suvarshachennareddy/minicni/common/constants"
	"github.com/suvarshachennareddy/minicni/common/proto"
	"github.com/suvarshachennareddy/minicni/common/types"
)

type Manager interface {
	Handle(ctx context.Context, args *proto.CNIArgs) error
}

type manager struct {
	netConfPath      string
	pluginBinPath    string
	attachmentStates map[types.AttachmentKey]types.AttachmentBody
	mu               sync.Mutex
}

func New(netConfPath string, pluginBinPath string) (Manager, error) {
	if netConfPath == "" {
		return nil, errors.New("netConfPath is empty")
	}

	_, err := os.Stat(netConfPath)
	if err != nil {
		return nil, err
	}

	if pluginBinPath == "" {
		return nil, errors.New("pluginBinPath is empty")
	}

	_, err = os.Stat(pluginBinPath)
	if err != nil {
		return nil, err
	}

	return &manager{
		netConfPath:      netConfPath,
		pluginBinPath:    pluginBinPath,
		attachmentStates: make(map[types.AttachmentKey]types.AttachmentBody),
	}, nil
}

func (m *manager) Handle(ctx context.Context, args *proto.CNIArgs) error {
	// I could not find a strong enough example where a network namespace ID
	// isn't a one-to-one mapping to a container ID, which the spec defines
	// as a network isolation domain, on a single host using network namespaces.
	// Ref: https://www.cni.dev/docs/spec/
	containerID, err := m.deriveContainerID(args.NetNsPath)
	if err != nil {
		return fmt.Errorf("could not derive container ID: %v", err)
	}

	netConfig, err := m.findNetworkConfig(args.NetworkName)
	if err != nil {
		return fmt.Errorf("could not fetch network config: %v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	switch args.CniCommand {
	case types.CNIAdd:
		return m.handleAdd(netConfig, containerID, args)
	case types.CNIDel:
		return m.handleDel(netConfig, containerID, args)
	case types.CNICheck:
		return m.handleCheck(netConfig, containerID, args)
	default:
		return fmt.Errorf("unsupported CNI command")
	}
}

func (m *manager) callPlugin(input []byte, pluginType string, containerID string, args *proto.CNIArgs) ([]byte, error) {
	cmd := exec.Command(m.pluginBinPath + "/" + pluginType)
	cmd.Env = append(cmd.Env,
		"CNI_COMMAND="+string(args.CniCommand),
		"CNI_CONTAINERID="+containerID,
		"CNI_NETNS="+args.NetNsPath,
		"CNI_IFNAME="+args.InterfaceName,
		constants.PluginBinPathEnv+"="+m.pluginBinPath,
	)

	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("exit error %v: %s", err, exitErr.Stderr)
		}

		return nil, err
	}

	return output, nil
}

func (m *manager) handleAdd(
	netConfig *types.NetworkConfig,
	containerID string,
	args *proto.CNIArgs,
) error {
	var requestConfig []byte
	var prevResult *types.AttachmentBody

	for i := range netConfig.Plugins {
		var err error

		requestConfigBuilder := NewRequestConfigBuilder().
			CniVersion(constants.CniVersion).
			NetworkName(netConfig.Name).
			Plugin(netConfig.Plugins[i], args.CapabilityArgs).
			PrevResult(prevResult)

		pluginType, err := requestConfigBuilder.ExtractPluginType()
		if err != nil {
			return fmt.Errorf("could not extract plugin type: %v", err)
		}

		requestConfig, err = requestConfigBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build request config for plugin: %v", err)
		}

		output, err := m.callPlugin(requestConfig, pluginType, containerID, args)
		if err != nil {
			return fmt.Errorf("could not successfully call plugin: %v", err)
		}

		attachmentBody := &types.AttachmentBody{}
		err = json.Unmarshal(output, attachmentBody)
		if err != nil {
			return fmt.Errorf("could not unmarshal plugin result: %v", err)
		}

		prevResult = attachmentBody
	}

	attachmentkey := types.AttachmentKey{
		ContainerID: containerID,
		IfName:      args.InterfaceName,
	}
	m.attachmentStates[attachmentkey] = *prevResult

	return nil
}

func (m *manager) handleDel(
	netConfig *types.NetworkConfig,
	containerID string,
	args *proto.CNIArgs,
) error {
	var requestConfig []byte

	attachmentkey := types.AttachmentKey{
		ContainerID: containerID,
		IfName:      args.InterfaceName,
	}
	prevResult, found := m.attachmentStates[attachmentkey]
	if !found {
		return nil
	}

	numPlugins := len(netConfig.Plugins)

	for i := range netConfig.Plugins {
		var err error

		requestConfigBuilder := NewRequestConfigBuilder().
			CniVersion(constants.CniVersion).
			NetworkName(netConfig.Name).
			Plugin(netConfig.Plugins[numPlugins-i-1], args.CapabilityArgs).
			PrevResult(&prevResult)

		pluginType, err := requestConfigBuilder.ExtractPluginType()
		if err != nil {
			return fmt.Errorf("could not extract plugin type: %v", err)
		}

		requestConfig, err = requestConfigBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build request config for plugin: %v", err)
		}

		_, err = m.callPlugin(requestConfig, pluginType, containerID, args)
		if err != nil {
			return fmt.Errorf("could not successfully call plugin: %v", err)
		}
	}

	delete(m.attachmentStates, attachmentkey)

	return nil
}

func (m *manager) handleCheck(
	netConfig *types.NetworkConfig,
	containerID string,
	args *proto.CNIArgs,
) error {
	var requestConfig []byte

	attachmentkey := types.AttachmentKey{
		ContainerID: containerID,
		IfName:      args.InterfaceName,
	}
	prevResult, found := m.attachmentStates[attachmentkey]
	if !found {
		return fmt.Errorf("no attachment found for container %s, interface %s", containerID, args.InterfaceName)
	}

	for i := range netConfig.Plugins {
		var err error

		requestConfigBuilder := NewRequestConfigBuilder().
			CniVersion(constants.CniVersion).
			NetworkName(netConfig.Name).
			Plugin(netConfig.Plugins[i], args.CapabilityArgs).
			PrevResult(&prevResult)

		pluginType, err := requestConfigBuilder.ExtractPluginType()
		if err != nil {
			return fmt.Errorf("could not extract plugin type: %v", err)
		}

		requestConfig, err = requestConfigBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build request config for plugin: %v", err)
		}

		_, err = m.callPlugin(requestConfig, pluginType, containerID, args)
		if err != nil {
			return fmt.Errorf("could not successfully call plugin: %v", err)
		}
	}

	return nil
}

func (m *manager) deriveContainerID(netNsPath string) (string, error) {
	inode, err := getInode(netNsPath)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(inode, 10), nil
}

func (m *manager) findNetworkConfig(network string) (*types.NetworkConfig, error) {
	networkConfig := &types.NetworkConfig{}

	err := filepath.WalkDir(m.netConfPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			if path == m.netConfPath {
				return err
			}

			log.Printf("skipping %s: %v", path, err)

			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		networkConfig = isNetworkConfig(path, network)
		if networkConfig != nil {
			return filepath.SkipAll
		}

		return nil
	})

	if networkConfig == nil {
		err = errors.Join(err, NetworkConfigNotFound)
	}

	return networkConfig, err
}
