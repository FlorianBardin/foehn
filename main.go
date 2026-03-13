package main

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-git/go-git/v6"
)

func main() {
	dirPath := "./tmp/build1"
	repoURL := "https://github.com/mmumshad/simple-webapp-docker"
	archivePath := "./archive/build1.tar"
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := cli.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = cloneRepo(dirPath, repoURL)
	if err != nil {
		log.Fatal(err)
	}

	err = toArchive(archivePath, dirPath)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	res, err := cli.ImageBuild(ctx, f, build.ImageBuildOptions{Tags: []string{"app1:latest"}})
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}

	config := &container.Config{
		Image:        "app1:latest",
		ExposedPorts: nat.PortSet{"8080/tcp": {}},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
		},
	}

	createResponse, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "my-app1")
	if err != nil {
		log.Fatal(err)
	}

	err = cli.ContainerStart(ctx, createResponse.ID, container.StartOptions{})
	if err != nil {
		log.Fatal(err)
	}
}

func toArchive(archivePath string, dirPath string) error {
	tarFile, err := os.Create(archivePath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := tarFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	tw := tar.NewWriter(tarFile)
	defer func() {
		err := tw.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = filepath.Walk(dirPath, generateWalkFunc(dirPath, tw))
	if err != nil {
		return err
	}
	return nil
}

func generateWalkFunc(dirPath string, tw *tar.Writer) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)

		if name == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		hdr.Name = name
		if info.IsDir() {
			hdr.Name += "/"
		}

		fmt.Printf("%+v\n", hdr)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err = io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	}
}

func cloneRepo(dirPath string, repoURL string) error {
	_, err := git.PlainClone(dirPath, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err = os.RemoveAll(dirPath + "/.git"); err != nil {
		log.Fatal(err)
	}
	return err
}
