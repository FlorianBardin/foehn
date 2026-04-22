package main

import (
	"net/http"
)

func main() {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:    ":4805",
		Handler: router,
	}

	err := server.ListenAndServe()
	if err != nil {
		return
	}
}
