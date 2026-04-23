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
	"github.com/google/uuid"
	"github.com/moby/go-archive"
)

type ContainerInfo struct {
	Name       string
	ID         string
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

	imageDetails, err := cli.ImageInspect(ctx, containerName)
	if err != nil {
		return ContainerInfo{}, err
	}

	if len(imageDetails.Config.ExposedPorts) == 0 {
		return ContainerInfo{}, fmt.Errorf("deployement refused: dockerfile should contains at least one exposed port")
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
		Image:        containerName + ":latest",
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
		return ContainerInfo{}, err
	}

	inspectResponse, err := cli.ContainerInspect(ctx, creationResponse.ID)
	if err != nil {
		return ContainerInfo{}, err
	}

	if len(inspectResponse.NetworkSettings.Ports[containerPort]) == 0 {
		return ContainerInfo{}, fmt.Errorf("deployement refused: container should at least exposed one port")
	}
	publicPort := inspectResponse.NetworkSettings.Ports[containerPort][0].HostPort

	return ContainerInfo{containerName, creationResponse.ID, publicPort}, nil
}

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

func RemoveContainer(cli *client.Client, ctx context.Context, containerID string) error {
	err := cli.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		return err
	}
	err = cli.ContainerRemove(ctx, containerID, container.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

func RemoveImage(cli *client.Client, ctx context.Context, imageID string) error {
	_, err := cli.ImageRemove(ctx, imageID, image.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}
