package go_base

import (
	"context"
	_ "go-base/db"
	_ "go-base/redis"
	"testing"
)

func TestBoot(t *testing.T) {
	// 注册监听器
	Bootstrap(context.Background(), "config.yml")

}
