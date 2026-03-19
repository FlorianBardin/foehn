package builder

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/moby/go-archive"
)

type ContainerInfo struct {
	Name       string
	PublicPort string
}

func BuildAndRun(cli *client.Client, ctx context.Context, dirPath string) (ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	containerName := "foehn-app-" + uuid.New().String()

	readArchive, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return ContainerInfo{}, err
	}
	defer readArchive.Close()

	res, err := cli.ImageBuild(ctx, readArchive, build.ImageBuildOptions{Tags: []string{containerName + ":latest"}})
	if err != nil {
		return ContainerInfo{}, err
	}
	defer res.Body.Close()

	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		return ContainerInfo{}, err
	}

	config := &container.Config{
		Image:        containerName + ":latest",
		ExposedPorts: nat.PortSet{"8080/tcp": {}},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
		},
	}

	creationResponse, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return ContainerInfo{}, err
	}

	err = cli.ContainerStart(ctx, creationResponse.ID, container.StartOptions{})
	if err != nil {
		return ContainerInfo{}, err
	}

	inspectResponse, err := cli.ContainerInspect(ctx, creationResponse.ID)
	if err != nil {
		return ContainerInfo{}, err
	}

	var publicPort string
	for _, ports := range inspectResponse.NetworkSettings.Ports {
		publicPort = ports[0].HostPort
		break
	}

	return ContainerInfo{containerName, publicPort}, nil
}
