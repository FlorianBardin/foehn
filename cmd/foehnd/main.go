package main

import (
	"log"
	"net/http"

	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"
)

func main() {
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

	router := http.NewServeMux()

	server := &http.Server{
		Addr:    ":4805",
		Handler: router,
	}

	err = server.ListenAndServe()
	if err != nil {
		return
	}
}
