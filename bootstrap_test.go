package go_base

import (
	"context"
	"go-base/config"
	_ "go-base/db"
	_ "go-base/redis"
	_ "go-base/web"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBoot(t *testing.T) {
	var api = []config.WebGroup{
		{Path: "/api", WebPaths: []config.WebPath{
			{Path: "/hello", Method: http.MethodGet, Handler: func(c *gin.Context) { c.String(http.StatusOK, "UP") }},
		}},
	}
	Bootstrap(context.Background(), "config.yml", config.WithWebApi(api))
}
