package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/moby/go-archive"
)

type ContainerInfo struct {
	Name       string
	PublicPort string
}

// BuildAndRun builds a Docker image from a project archive whose path
// is passed as an argument. The project must contain a Dockerfile that
// exposes at least one port. Using this image, it then launches a container
// and maps the container’s selected port to a port on the host machine.
// BuildAndRun then returns information about the created container.
func BuildAndRun(cli *client.Client, ctx context.Context, dirPath string, appID string) (ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	imageName := "foehn-image-" + appID
	containerName := "foehn-app-" + appID

	readArchive, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return ContainerInfo{}, err
	}
	defer readArchive.Close()

	res, err := cli.ImageBuild(ctx, readArchive, build.ImageBuildOptions{Tags: []string{imageName + ":latest"}})
	if err != nil {
		return ContainerInfo{}, err
	}
	defer res.Body.Close()

	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		return ContainerInfo{}, err
	}

	imageDetails, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		return ContainerInfo{}, err
	}

	if len(imageDetails.Config.ExposedPorts) == 0 {
		return ContainerInfo{}, fmt.Errorf("deployment refused: dockerfile should contains at least one exposed port")
	}

	for k, v := range imageDetails.Config.ExposedPorts {
		fmt.Printf("exposed port: %v with key : %v \n", v, k)
	}

	port, err := extractWebPort(imageDetails.Config.ExposedPorts)
	if err != nil {
		return ContainerInfo{}, err
	}

	containerPort := nat.Port(port + "/tcp")

	config := &container.Config{
		Image:        imageName + ":latest",
		ExposedPorts: nat.PortSet{containerPort: {}},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
		},
	}

	creationResponse, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return ContainerInfo{}, err
	}

	err = cli.ContainerStart(ctx, creationResponse.ID, container.StartOptions{})
	if err != nil {
		_ = StopAndRemoveContainer(ctx, cli, appID)
		return ContainerInfo{}, err
	}

	inspectResponse, err := cli.ContainerInspect(ctx, creationResponse.ID)
	if err != nil {
		_ = StopAndRemoveContainer(ctx, cli, appID)
		return ContainerInfo{}, err
	}

	if len(inspectResponse.NetworkSettings.Ports[containerPort]) == 0 {
		_ = StopAndRemoveContainer(ctx, cli, appID)
		return ContainerInfo{}, fmt.Errorf("deployment refused: container should at least exposed one port")
	}
	publicPort := inspectResponse.NetworkSettings.Ports[containerPort][0].HostPort

	return ContainerInfo{containerName, publicPort}, nil
}

// extractWebPort returns the most suitable port from a list of exposed ports in a
// Docker image passed as an argument. extractWebPort first retrieves all the exposed ports,
// then compares them to the following list of priority ports in the same order: 80, 8080, 3000,
// 5173, 8000, 443, 8443, 5000. If a priority port matches, extractWebPort returns it. Otherwise,
// it returns the lowest-numbered exposed port.
func extractWebPort(exposedPorts map[string]struct{}) (lowestPort string, err error) {
	var ports []int
	priorityPorts := []int{80, 8080, 3000, 5173, 8000, 443, 8443, 5000}

	for port := range exposedPorts {
		intPort, cvrtError := strconv.Atoi(strings.Split(port, "/")[0])
		if cvrtError != nil {
			cvrtError = fmt.Errorf("failed to convert port %s to int : %v", port, cvrtError)
			err = errors.Join(err, cvrtError)
			continue
		}
		ports = append(ports, intPort)
	}

	for _, priorityPort := range priorityPorts {
		if slices.Contains(ports, priorityPort) {
			return strconv.Itoa(priorityPort), err
		}
	}

	sort.Ints(ports)

	if len(ports) == 0 {
		err = errors.Join(err, fmt.Errorf("no valid port found"))
		return "", err
	}

	lowestPort = strconv.Itoa(ports[0])

	return lowestPort, err
}

// StopAndRemoveContainer stops and removes a container based on its appID, which is defined during deployment.
func StopAndRemoveContainer(ctx context.Context, cli *client.Client, appID string) error {
	err := cli.ContainerStop(ctx, "foehn-app-"+appID, container.StopOptions{})
	if err != nil {
		return err
	}
	err = cli.ContainerRemove(ctx, "foehn-app-"+appID, container.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

// RemoveImage removes a Docker image by its ID.
func RemoveImage(ctx context.Context, cli *client.Client, imageID string) error {
	_, err := cli.ImageRemove(ctx, imageID, image.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}
