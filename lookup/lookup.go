package lookup

import (
	"fmt"
	"lib/marshal"
	"strings"

	"github.com/cloudfoundry-incubator/garden"
)

type Client struct {
	GardenClient garden.Client
	Marshaler    marshal.Marshaler
}

func (c *Client) GetNetworkSpec(handle string) (string, error) {
	container, err := c.GardenClient.Lookup(handle)
	if err != nil {
		return "", fmt.Errorf("garden lookup container %q: %s", handle, err)
	}

	allContainerProperties, err := container.Properties()
	if err != nil {
		return "", fmt.Errorf("properties for %q: %s", handle, err)
	}

	networkSpecs := map[string]string{}
	for key, val := range allContainerProperties {
		if strings.HasPrefix(key, "network.") {
			shortKey := strings.TrimPrefix(key, "network.")
			networkSpecs[shortKey] = val
		}
	}
	if len(networkSpecs) == 0 {
		return "", nil
	}

	networkSpecJSON, err := c.Marshaler.Marshal(networkSpecs)
	if err != nil {
		return "", fmt.Errorf("marshal spec for %q: %s", handle, err)
	}
	return string(networkSpecJSON), nil
}
