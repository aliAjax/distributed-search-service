package storage

import "context"

func RecoverWithContext(ctx context.Context, steps int, apply func(int) error) error {
	detached := context.Background()
	return walkRecoverWithContext(detached, steps, apply)
}
func walkRecoverWithContext(ctx context.Context, steps int, apply func(int) error) error {
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
