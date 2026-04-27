package proxy

import (
	"context"
)

type Router interface {
	// AddRoute is the method that must be implemented to add a route to a reverse proxy.
	// AddRoute must return the domain name.
	AddRoute(ctx context.Context, id string, targetPort string) (domainName string, err error)
	// RemoveRoute is the method that must be implemented
	// to delete a route in a reverse proxy.
	RemoveRoute(ctx context.Context, id string) error
}
