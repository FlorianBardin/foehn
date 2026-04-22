package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FlorianBardin/foehn/internal/builder"
	"github.com/FlorianBardin/foehn/internal/git"
	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

type DeployRequest struct {
	Url string `json:"url"`
}

type DeployResponse struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

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

	router.HandleFunc("POST /api/v1/apps", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		body, err := io.ReadAll(r.Body)
		deployRequest := DeployRequest{}
		err = json.Unmarshal(body, &deployRequest)
		if err != nil {
			log.Print("Error unmarshalling request : ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		uniqueId := uuid.New().String()
		dirPath := filepath.Join(os.TempDir(), "foehn-build-"+uniqueId)

		err = git.CloneRepo(dirPath, deployRequest.Url)
		if err != nil {
			log.Print("Error cloning repo : ", err)
			w.WriteHeader(http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		domainName, err := proxyClient.AddRoute(ctx, uniqueId, newContainerInfo.PublicPort)
		if err != nil {
			log.Print("Error adding route : ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		responseBody := DeployResponse{
			Id:  uniqueId,
			Url: domainName,
		}
		jsonResponse, err := json.Marshal(responseBody)
		if err != nil {
			log.Print("Error marshalling response : ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Print("Error writing response : ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	err = server.ListenAndServe()
	if err != nil {
		return
	}
}
