package proxy

import (
	"context"
)

type Router interface {
	AddRoute(ctx context.Context, domainName string, targetPort string) error
	RemoveRoute(ctx context.Context, domainName string) error
}
