package proxy

import (
	"context"
)

type Router interface {
	AddRoute(ctx context.Context, id string, targetPort string) (domainName string, err error)
	RemoveRoute(ctx context.Context, id string) error
}
