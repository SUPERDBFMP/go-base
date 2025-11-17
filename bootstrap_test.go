package go_base

import (
	"context"
	"testing"
)

func TestBoot(t *testing.T) {
	Bootstrap(context.Background(), "config.yml")
}
