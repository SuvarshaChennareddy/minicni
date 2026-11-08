package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindNetworkConfig(t *testing.T) {
	m := &manager{netConfPath: "./testing"}

	networkConfig, err := m.findNetworkConfig("dbnet")
	require.NoError(t, err)
	require.NotNil(t, networkConfig)

	assert.Equal(t, "1.1.0", networkConfig.CniVersion)
	assert.Equal(t, []string{"0.3.1", "0.4.0", "1.0.0", "1.1.0"}, networkConfig.CniVersions)
	assert.Equal(t, "dbnet", networkConfig.Name)

	wantPlugins := []string{
		`{
			"type": "bridge",
			"bridge": "cni0",
			"keyA": ["some more", "plugin specific", "configuration"],
			"ipam": {
				"type": "host-local",
				"subnet": "10.1.0.0/16",
				"gateway": "10.1.0.1",
				"routes": [
					{"dst": "0.0.0.0/0"}
				]
			},
			"dns": {
				"nameservers": ["10.1.0.1"]
			}
		}`,
		`{
			"type": "tuning",
			"capabilities": {
				"mac": true
			},
			"sysctl": {
				"net.core.somaxconn": "500"
			}
		}`,
		`{
			"type": "portmap",
			"capabilities": {"portMappings": true}
		}`,
	}

	require.Len(t, networkConfig.Plugins, len(wantPlugins))
	for i, want := range wantPlugins {
		// RawMessage keeps the file's whitespace, so compare as JSON, not bytes.
		assert.JSONEq(t, want, string(networkConfig.Plugins[i]), "plugin %d", i)
	}
}
