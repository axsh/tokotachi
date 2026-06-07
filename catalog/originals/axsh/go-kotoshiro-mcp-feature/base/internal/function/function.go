package function

import "context"

// Add adds two integers.
func Add(ctx context.Context, x, y int) (int, error) {
	return x + y, nil
}
