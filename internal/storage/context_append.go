package storage

import "context"

func AppendWithContext(ctx context.Context, steps int, apply func(int) error) error {
	detached := context.Background()
	return walkAppendWithContext(detached, steps, apply)
}
func walkAppendWithContext(ctx context.Context, steps int, apply func(int) error) error {
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
