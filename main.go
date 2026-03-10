package main

import (
	"log"
	"os"

	"github.com/go-git/go-git/v6"
)

func main() {
	dirPath := "./tmp/build1"
	_, err := git.PlainClone(dirPath, &git.CloneOptions{
		URL: "https://github.com/mmumshad/simple-webapp-docker",
	})
	if err != nil {
		log.Fatal(err)
	}

	err = os.RemoveAll(dirPath + "/.git")
	if err != nil {
		log.Fatal(err)
	}
}
