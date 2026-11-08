package manager

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRuntimeConfigMap(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()

	runtimeConfigMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(raw, &runtimeConfigMap); err != nil {
		t.Fatalf("failed to make runtimeConfigMap: %v", err)
	}

	return runtimeConfigMap
}

func TestRequestConfigBuilderPlugin(t *testing.T) {
	runtimeConfigMap := makeRuntimeConfigMap(t, []byte(`{
		"mac": "00:11:22:33:44:66",
		"portMappings": [
			{"hostPort": 8080, "containerPort": 80, "protocol": "tcp"}
		]
	}`))

	tests := []struct {
		name         string
		pluginConfig string
		want         string
	}{
		{
			name: "no capabilities passes through unchanged",
			pluginConfig: `{
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
			want: `{
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
		},
		{
			name: "declared capability becomes runtimeConfig",
			pluginConfig: `{
				"type": "tuning",
				"capabilities": {
					"mac": true
				},
				"sysctl": {
					"net.core.somaxconn": "500"
				}
			}`,
			want: `{
				"type": "tuning",
				"runtimeConfig": {
					"mac": "00:11:22:33:44:66"
				},
				"sysctl": {
					"net.core.somaxconn": "500"
				}
			}`,
		},
		{
			name: "capability set to false is not passed",
			pluginConfig: `{
				"type": "portmap",
				"capabilities": {"portMappings": false}
			}`,
			want: `{
				"type": "portmap"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRequestConfigBuilder().
				Plugin(json.RawMessage(tt.pluginConfig), runtimeConfigMap).
				Build()
			require.NoError(t, err)

			assert.JSONEq(t, tt.want, string(got))
		})
	}
}
