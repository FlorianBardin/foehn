package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/docker/go-connections/nat"
	"github.com/go-git/go-git/v6"
	"github.com/google/uuid"
	"github.com/moby/go-archive"
)

func main() {
	uniqueId := uuid.New().String()
	baseTmpDir := os.TempDir()
	dirPath := filepath.Join(baseTmpDir, "foehn-build-"+uniqueId)
	repoURL := "https://github.com/FlorianBardin/simple-webapp-docker"

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Print("Error creating docker client : ", err)
		return
	}
	defer func() {
		err := cli.Close()
		if err != nil {
			log.Print("Error closing docker client : ", err)
		}
	}()

	err = cloneRepo(dirPath, repoURL)
	if err != nil {
		log.Print("Error cloning repo : ", err)
		return
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Print("Failed to remove directory : ", path)
		}
	}(dirPath)

	readArchive, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return
	}
	defer func(readArchive io.ReadCloser) {
		err := readArchive.Close()
		if err != nil {
			log.Print("Error closing archive : ", err)
		}
	}(readArchive)

	err = buildAndRun(cli, ctx, readArchive)
	if err != nil {
		log.Print("Failed to build and run from tar : ", err)
	}
}

func buildAndRun(cli *client.Client, ctx context.Context, r io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	containerName := "foehn-app-" + uuid.New().String()

	res, err := cli.ImageBuild(ctx, r, build.ImageBuildOptions{Tags: []string{containerName + ":latest"}})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		return err
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
		return err
	}

	err = cli.ContainerStart(ctx, creationResponse.ID, container.StartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func cloneRepo(dirPath string, repoURL string) error {
	_, err := git.PlainClone(dirPath, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return err
	}

	if err = os.RemoveAll(filepath.Join(dirPath, ".git")); err != nil {
		return err
	}

	return nil
}
