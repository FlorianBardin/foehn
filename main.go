package main

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/go-git/go-git/v6"
)

func main() {
	dirPath := "./tmp/build1"
	repoURL := "https://github.com/FlorianBardin/florianbardin-portfolio"
	archivePath := "./archive/build1.tar"

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
		return
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
