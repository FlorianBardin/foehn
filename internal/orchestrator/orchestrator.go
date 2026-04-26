package orchestrator

import (
	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"
)

type Orchestrator struct {
	dockerClient *client.Client
	proxyClient  proxy.Router
}

func NewOrchestrator(dockerClient *client.Client, proxyClient proxy.Router) *Orchestrator {
	return &Orchestrator{
		dockerClient: dockerClient,
		proxyClient:  proxyClient,
	}
}
