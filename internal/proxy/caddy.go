package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	ID     string        `json:"@id"`
	Match  []CaddyMatch  `json:"match"`
	Handle []CaddyHandle `json:"handle"`
}

type CaddyClient struct {
	apiURL string
	client *http.Client
}

func (c *CaddyClient) AddRoute(ctx context.Context, id string, targetPort string) (domainName string, err error) {
	domainName = id + ".foehn.localhost"

	newRoute := CaddyRoute{
		ID: id,
		Match: []CaddyMatch{
			{
				Host: []string{domainName},
			},
		},
		Handle: []CaddyHandle{
			{
				Handler: "reverse_proxy",
				Upstreams: []Upstream{
					{
						Dial: "127.0.0.1:" + targetPort,
					},
				},
			},
		},
	}

	newRouteJson, err := json.Marshal(newRoute)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		"PUT",
		c.apiURL+"/config/apps/http/servers/srv0/routes/0",
		bytes.NewReader(newRouteJson))
	fmt.Println(c.apiURL + "/config/apps/http/servers/srv0/routes/0")
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(request)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s", body)
	}

	return domainName, nil
}

func (c *CaddyClient) RemoveRoute(ctx context.Context, domainName string) error {
	return nil
}

func NewCaddyClient(apiURL string) Router {
	return &CaddyClient{apiURL: apiURL, client: &http.Client{}}
}
