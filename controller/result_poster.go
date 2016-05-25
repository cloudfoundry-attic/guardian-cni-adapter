package controller

import (
	"fmt"
	"net/http"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/dghubble/sling"
)

type rp struct {
	slingClient *sling.Sling
}

func NewResultPoster(baseURL string, httpClient *http.Client) *rp {
	slingClient := sling.New().Client(httpClient).Base(baseURL).Set("Accept", "application/json")
	return &rp{
		slingClient: slingClient,
	}
}

func (c *rp) AddResult(containerID string, netConfig *libcni.NetworkConfig, result *types.Result) error {
	payload := struct {
		ContainerID string
		Config      *libcni.NetworkConfig
		Result      *types.Result
	}{
		ContainerID: containerID,
		Config:      netConfig,
		Result:      result,
	}
	resp, err := c.slingClient.New().Post("/cni-result/add").BodyJSON(payload).Receive(nil, nil)
	if err != nil {
		return fmt.Errorf("add result: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add result: unexpected status code: %s", resp.Status)
	}
	return nil
}

func (c *rp) DelResult(containerID string, netConfig *libcni.NetworkConfig) error {
	payload := struct {
		ContainerID string
		Config      *libcni.NetworkConfig
	}{
		ContainerID: containerID,
		Config:      netConfig,
	}
	resp, err := c.slingClient.New().Post("/cni-result/del").BodyJSON(payload).Receive(nil, nil)
	if err != nil {
		return fmt.Errorf("del result: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("del result: unexpected status code: %s", resp.Status)
	}
	return nil
}
