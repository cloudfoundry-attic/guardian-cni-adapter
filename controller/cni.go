package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/appc/cni/libcni"
)

type CNIController struct {
	PluginDir string
	ConfigDir string

	CNIConfig      *libcni.CNIConfig
	NetworkConfigs []*libcni.NetworkConfig
}

func (c *CNIController) ensureInitialized() error {
	if c.CNIConfig == nil {
		c.CNIConfig = &libcni.CNIConfig{Path: []string{c.PluginDir}}
	}

	if c.NetworkConfigs == nil {
		c.NetworkConfigs = []*libcni.NetworkConfig{}

		err := filepath.Walk(c.ConfigDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".conf") {
				return nil
			}

			conf, err := libcni.ConfFromFile(path)
			if err != nil {
				return fmt.Errorf("unable to load config from %s: %s", path, err)
			}
			c.NetworkConfigs = append(c.NetworkConfigs, conf)
			log.Printf("loaded config %+v\n%s\n", conf.Network, string(conf.Bytes))
			return nil
		})
		if err != nil {
			return fmt.Errorf("error loading config: %s", err)
		}
	}

	return nil
}

func isCIDR(spec string) bool {
	_, _, err := net.ParseCIDR(spec)
	return err == nil
}

func isIP(spec string) bool {
	ip := net.ParseIP(spec)
	return ip != nil
}

func AppendNetworkSpec(existingNetConfig *libcni.NetworkConfig, gardenNetworkSpec string) (*libcni.NetworkConfig, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(existingNetConfig.Bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %s", err)
	}

	if gardenNetworkSpec != "" && !isCIDR(gardenNetworkSpec) && !isIP(gardenNetworkSpec) {
		networkPayloadMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(gardenNetworkSpec), &networkPayloadMap)
		if err != nil {
			return nil, fmt.Errorf("unmarshal garden network spec: %s", err)
		}
		config["network"] = networkPayloadMap
	}

	newBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err //Not tested
	}

	return &libcni.NetworkConfig{
		Network: existingNetConfig.Network,
		Bytes:   newBytes,
	}, nil
}

func (c *CNIController) Up(namespacePath, handle, spec string) error {
	err := c.ensureInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %s", err)
	}

	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := AppendNetworkSpec(networkConfig, spec)
		if err != nil {
			return fmt.Errorf("adding garden network spec to CNI config: %s", err)
		}

		result, err := c.CNIConfig.AddNetwork(enhancedNetConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("add network failed: %s", err)
		}

		log.Printf("up result for name=%s, type=%s: \n%s\n", networkConfig.Network.Name, networkConfig.Network.Type, result.String())
	}

	return nil
}

func (c *CNIController) Down(namespacePath, handle string) error {
	err := c.ensureInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %s", err)
	}

	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}
		err = c.CNIConfig.DelNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}
		log.Printf("down complete for name=%s, type=%s\n", networkConfig.Network.Name, networkConfig.Network.Type)
	}

	return nil
}
