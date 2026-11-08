package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/suvarshachennareddy/minicni/common/types"
)

type RequestConfigBuilder struct {
	requestConfig map[string]json.RawMessage
	err           error
}

func NewRequestConfigBuilder() *RequestConfigBuilder {
	b := &RequestConfigBuilder{}

	b.requestConfig = make(map[string]json.RawMessage)

	return b
}

func (b *RequestConfigBuilder) addField(key string, value json.RawMessage) {
	b.requestConfig[key] = value
}

func (b *RequestConfigBuilder) CniVersion(version string) *RequestConfigBuilder {
	raw, err := json.Marshal(version)
	if err != nil {
		b.err = errors.Join(b.err, err)

		return b
	}

	b.addField("cniVersion", raw)

	return b
}

func (b *RequestConfigBuilder) NetworkName(name string) *RequestConfigBuilder {
	raw, err := json.Marshal(name)
	if err != nil {
		b.err = errors.Join(b.err, err)

		return b
	}

	b.addField("name", raw)

	return b
}

func (b *RequestConfigBuilder) Plugin(
	pluginConfig json.RawMessage,
	runtimeConfigMap map[string]json.RawMessage,
) *RequestConfigBuilder {
	pluginConfigMap := make(map[string]json.RawMessage)
	err := json.Unmarshal(pluginConfig, &pluginConfigMap)
	if err != nil {
		b.err = errors.Join(b.err, err)

		return b
	}

	capabilities, foundCapabilities := pluginConfigMap["capabilities"]
	delete(pluginConfigMap, "capabilities")

	if runtimeConfigMap != nil && foundCapabilities {
		capabiltiesMap := make(map[string]bool)

		err := json.Unmarshal(capabilities, &capabiltiesMap)
		if err != nil {
			b.err = errors.Join(b.err, err)

			return b
		}

		pluginRuntimeConfigMap := make(map[string]json.RawMessage)

		for capability, want := range capabiltiesMap {
			if !want {
				continue
			}

			value, found := runtimeConfigMap[capability]
			if found {
				pluginRuntimeConfigMap[capability] = value
			}
		}

		if len(pluginRuntimeConfigMap) > 0 {
			pluginRuntimeConfig, err := json.Marshal(pluginRuntimeConfigMap)
			if err != nil {
				b.err = errors.Join(b.err, err)

				return b
			}

			pluginConfigMap["runtimeConfig"] = pluginRuntimeConfig
		}

	}

	maps.Copy(b.requestConfig, pluginConfigMap)

	return b
}

func (b *RequestConfigBuilder) PrevResult(attachment *types.AttachmentBody) *RequestConfigBuilder {
	if attachment == nil {
		return b
	}

	raw, err := json.Marshal(attachment)
	if err != nil {
		b.err = errors.Join(b.err, err)

		return b
	}

	b.addField("prevResult", raw)

	return b
}

func (b *RequestConfigBuilder) ExtractPluginType() (string, error) {
	var pluginType string

	err := json.Unmarshal(b.requestConfig["type"], &pluginType)
	if err != nil {
		return "", err
	}

	if pluginType == "" {
		return "", fmt.Errorf("plugin type is empty")
	}

	return pluginType, nil
}

func (b *RequestConfigBuilder) Build() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	return json.Marshal(b.requestConfig)
}
