package proxy

import (
	"context"
	"net/http"
)

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
