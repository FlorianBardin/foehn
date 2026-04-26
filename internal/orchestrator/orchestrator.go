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
	DomainName string
	ID         string
}

// NewOrchestrator creates a new orchestrator using the address of a proxy client and a Docker client.
// NewOrchestrator returns the address of this new orchestrator.
func NewOrchestrator(dockerClient *client.Client, proxyClient proxy.Router) *Orchestrator {
	return &Orchestrator{
		dockerClient: dockerClient,
		proxyClient:  proxyClient,
	}
}

// Deploy coordinates all stages of an application's deployment. It generates a unique appID
// and specifies the path where the project should be cloned (temporary directory), clones
// the directory using [git.CloneRepo], builds its image, and launches its container using [builder.BuildAndRun],
// then assigns it a unique domain name in the reverse proxy with [proxy.Router.AddRoute]. Deploy returns the information along
// with the domain name and appID. In the case of error, the container and the domain name are removed.
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

	return DeploymentInfo{DomainName: domainName, ID: appID}, nil
}

// Destroy removes the container whose appID is passed as an argument,
// using [builder.StopAndRemoveContainer]. It also removes the corresponding
// domain name from the reverse proxy with [proxy.Router.AddRoute].
func (o *Orchestrator) Destroy(ctx context.Context, appID string) error {
	containerErr := builder.StopAndRemoveContainer(ctx, o.dockerClient, appID)

	routeErr := o.proxyClient.RemoveRoute(ctx, appID)

	return errors.Join(containerErr, routeErr)
}
