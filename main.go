package main

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
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
)

func main() {
	uniqueId := uuid.New().String()
	baseTmpDir := os.TempDir()
	dirPath := filepath.Join(baseTmpDir, "foehn-build-"+uniqueId)
	archivePath := dirPath + ".tar"

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

	err = toArchive(archivePath, dirPath)
	if err != nil {
		log.Print("Error archiving : ", err)
		return
	}
	defer func(path string) {
		err := os.Remove(path)
		if err != nil {
			log.Print("Failed to remove directory : ", path)
		}
	}(archivePath)

	f, err := os.Open(archivePath)
	if err != nil {
		log.Print("Error opening archive : ", err)
		return
	}
	defer f.Close()

	err = buildAndRunFromTar(cli, ctx, f)
	if err != nil {
		log.Print("Failed to build and run from tar : ", err)
	}
}

func buildAndRunFromTar(cli *client.Client, ctx context.Context, f *os.File) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	containerName := "foehn-app-" + uuid.New().String()

	res, err := cli.ImageBuild(ctx, f, build.ImageBuildOptions{Tags: []string{containerName + ":latest"}})
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

func toArchive(archivePath string, dirPath string) (err error) {
	tarFile, e := os.Create(archivePath)
	if e != nil {
		return e
	}
	defer func() {
		closeErr := tarFile.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	tw := tar.NewWriter(tarFile)
	defer func() {
		closeErr := tw.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	err = filepath.Walk(dirPath, generateWalkFunc(dirPath, tw))

	return err
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
		return err
	}

	if err = os.RemoveAll(filepath.Join(dirPath, ".git")); err != nil {
		return err
	}

	return nil
}
