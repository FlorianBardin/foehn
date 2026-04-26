package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/FlorianBardin/foehn/internal/orchestrator"
	"github.com/FlorianBardin/foehn/internal/proxy"
	"github.com/docker/docker/client"
)

type DeployRequest struct {
	Url string `json:"url"`
}

type DeployResponse struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func main() {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Print("Error creating docker client : ", err)
		return
	}
	defer func() {
		err := dockerCli.Close()
		if err != nil {
			log.Print("Error closing docker client : ", err)
		}
	}()

	proxyCli := proxy.NewCaddyClient("http://localhost:2019")

	newOrchestrator := orchestrator.NewOrchestrator(dockerCli, proxyCli)

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

		deploymentInfo, err := newOrchestrator.Deploy(ctx, deployRequest.Url)
		if err != nil {
			log.Print("Error during deploy : ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		responseBody := DeployResponse{
			Id:  deploymentInfo.ID,
			Url: deploymentInfo.Url,
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

	router.HandleFunc("DELETE /api/v1/apps/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appID := r.PathValue("id")
		log.Printf("Deleting app id %s", appID)

		err := newOrchestrator.Destroy(ctx, appID)
		if err != nil {
			log.Print("Error during destroy : ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	err = server.ListenAndServe()
	if err != nil {
		return
	}
}
