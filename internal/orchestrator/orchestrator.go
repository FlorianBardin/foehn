package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/FlorianBardin/foehn/internal/builder"
	"github.com/FlorianBardin/foehn/internal/git"
	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type Orchestrator struct {
	dockerClient *client.Client
	proxyClient  proxy.Router
}

type DeploymentInfo struct {
	Url string
	ID  string
}

func NewOrchestrator(dockerClient *client.Client, proxyClient proxy.Router) *Orchestrator {
	return &Orchestrator{
		dockerClient: dockerClient,
		proxyClient:  proxyClient,
	}
}

func (o *Orchestrator) Deploy(ctx context.Context, repoUrl string) (deploymentInfo DeploymentInfo, err error) {
	appID := uuid.New().String()
	dirPath := filepath.Join(os.TempDir(), "foehn-build-"+appID)

	err = git.CloneRepo(dirPath, repoUrl)
	if err != nil {
		return DeploymentInfo{}, err
	}
	defer func(path string) {
		removeErr := os.RemoveAll(path)
		if removeErr != nil {
			err = errors.Join(err, removeErr)
		}
	}(dirPath)

	newContainerInfo, err := builder.BuildAndRun(o.dockerClient, ctx, dirPath, appID)
	if err != nil {
		return DeploymentInfo{}, err
	}

	domainName, err := o.proxyClient.AddRoute(ctx, appID, newContainerInfo.PublicPort)
	if err != nil {
		removeErr := builder.StopAndRemoveContainer(ctx, o.dockerClient, appID)
		if removeErr != nil {
			return DeploymentInfo{}, errors.Join(err, removeErr)
		}
		return DeploymentInfo{}, err
	}

	return DeploymentInfo{Url: domainName, ID: appID}, nil
}

func (o *Orchestrator) Destroy(ctx context.Context, appID string) error {
	containerErr := builder.StopAndRemoveContainer(ctx, o.dockerClient, appID)

	routeErr := o.proxyClient.RemoveRoute(ctx, appID)

	return errors.Join(containerErr, routeErr)
}
