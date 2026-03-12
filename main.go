package main

import (
	"log"
	"os"

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
