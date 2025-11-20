package go_base

import (
	"context"
	"net/http"
	"testing"

	"github.com/SUPERDBFMP/go-base/config"
	_ "github.com/SUPERDBFMP/go-base/db"
	_ "github.com/SUPERDBFMP/go-base/redis"
	_ "github.com/SUPERDBFMP/go-base/web"

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
