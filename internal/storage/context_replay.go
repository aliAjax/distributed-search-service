package storage

import "context"

func ReplayWithContext(ctx context.Context, steps int, apply func(int) error) error {
	return walkReplayWithContext(ctx, steps, apply)
}
func walkReplayWithContext(ctx context.Context, steps int, apply func(int) error) error {
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
