package function

import (
	"context"
	"testing"
)

func TestAdd(t *testing.T) {
	sum, err := Add(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum != 3 {
		t.Errorf("expected 3, got %d", sum)
	}
}
