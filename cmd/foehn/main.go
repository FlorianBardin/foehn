package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"

	"github.com/FlorianBardin/foehn/internal/builder"
	"github.com/FlorianBardin/foehn/internal/git"
	"github.com/google/uuid"
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

	proxyClient := proxy.NewCaddyClient("http://localhost:2019")

	err = git.CloneRepo(dirPath, repoURL)
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

	newContainerInfo, err := builder.BuildAndRun(cli, ctx, dirPath)
	if err != nil {
		log.Print("Failed to build and run from tar : ", err)
	}

	fmt.Println(newContainerInfo)

	domainName := "app-" + uniqueId + ".foehn.localhost"

	err = proxyClient.AddRoute(ctx, domainName, newContainerInfo.PublicPort)
	if err != nil {
		return
	}

	fmt.Println(domainName)
}
