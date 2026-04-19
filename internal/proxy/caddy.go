package proxy

import (
	"context"
	"net/http"
)

type CaddyMatch struct {
	Host []string `json:"host"`
}

type Upstream struct {
	Dial string `json:"dial"`
}

type CaddyHandle struct {
	Handler   string     `json:"handler"`
	Upstreams []Upstream `json:"upstreams"`
}

type CaddyRoute struct {
	Match  []CaddyMatch  `json:"match"`
	Handle []CaddyHandle `json:"handle"`
}

type CaddyClient struct {
	apiURL string
	client *http.Client
}

func (c *CaddyClient) AddRoute(ctx context.Context, domainName string, targetPort string) error {
	return nil
}

func (c *CaddyClient) RemoveRoute(ctx context.Context, domainName string) error {
	return nil
}

func NewCaddyClient(apiURL string) Router {
	return &CaddyClient{apiURL: apiURL, client: &http.Client{}}
}
