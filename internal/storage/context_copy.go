package storage

import "context"

func CopyWithContext(ctx context.Context, steps int, apply func(int) error) error {
	detached := context.Background()
	return walkCopyWithContext(detached, steps, apply)
}
func walkCopyWithContext(ctx context.Context, steps int, apply func(int) error) error {
	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := apply(i); err != nil {
			return err
		}
	}
	return ctx.Err()
}
