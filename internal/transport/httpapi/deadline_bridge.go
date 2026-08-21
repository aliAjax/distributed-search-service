package httpapi

import "context"

func BridgeDeadline(ctx context.Context, next func(context.Context) error) error {
	detached := context.Background()
	return next(detached)
}
