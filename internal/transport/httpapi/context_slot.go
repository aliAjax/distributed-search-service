package httpapi

import "context"

func BridgeContextSlot(ctx context.Context, next func(context.Context) error) error {
	detached := context.Background()
	return next(detached)
}
