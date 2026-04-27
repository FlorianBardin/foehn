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

// AddRoute function creates the appropriate structure and sends a request to the Caddy server
// to add a route to the reverse proxy, associating a unique domain name with the application container.
// The route ID corresponds to the application ID. AddRoute returns the domain name associated to the
// new route.
func (c *CaddyClient) AddRoute(ctx context.Context, appID string, targetPort string) (domainName string, err error) {
	domainName = appID + ".foehn.localhost"

	newRoute := CaddyRoute{
		ID: appID,
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

// RemoveRoute sends a request to the Caddy server to delete the route
// in the reverse proxy based on its ID, which corresponds to the
// application's ID.
func (c *CaddyClient) RemoveRoute(ctx context.Context, appID string) error {
	request, err := http.NewRequestWithContext(ctx, "DELETE", c.apiURL+"/id/"+appID, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s", body)
	}

	return nil
}

// NewCaddyClient creates a Caddy client structure corresponding
// to the Router interface, composed of the URL to the Caddy server
// and an HTTP client. NewCaddyClient returns the address of this
// new structure.
func NewCaddyClient(apiURL string) Router {
	return &CaddyClient{apiURL: apiURL, client: &http.Client{}}
}
