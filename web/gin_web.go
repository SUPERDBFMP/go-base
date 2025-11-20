package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/SUPERDBFMP/go-base/config"
	"github.com/SUPERDBFMP/go-base/glog"
	"github.com/SUPERDBFMP/go-base/listener"
	"github.com/SUPERDBFMP/go-base/metric"
	"github.com/SUPERDBFMP/go-base/util"
	"github.com/SUPERDBFMP/go-base/web/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// GinWebRouter Gin的web路由器
var GinWebRouter *gin.Engine

var httpServer *http.Server

func init() {
	listener.AddTypedApplicationListener(&AppConfigLoadedEventListener{})
	listener.AddTypedApplicationListener(&AppShutDownEventListener{})
}

type AppConfigLoadedEventListener struct{}

func (ace *AppConfigLoadedEventListener) GetOrder() int {
	return 9999
}

func (ace *AppConfigLoadedEventListener) OnApplicationEvent(ctx context.Context, event *listener.AppConfigLoadedEvent) {
	glog.Infof(ctx, "AppConfigLoadedEvent: %v", event.Time)
	if config.GlobalConf.WebServer != nil && event.BootstrapConfig != nil {
		InitWebServer(ctx, event.BootstrapConfig.WebApi, event.BootstrapConfig.WebValidators, event.BootstrapConfig.WebMiddlewares...)
	}
}

type AppShutDownEventListener struct{}

func (l *AppShutDownEventListener) GetOrder() int {
	return -9999
}

func (l *AppShutDownEventListener) OnApplicationEvent(ctx context.Context, event *listener.AppShutdownEvent) {
	glog.Infof(ctx, "开始关闭httpServer")
	// 关闭httpServer
	if httpServer != nil {
		// 设置优雅停机超时时间
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			glog.Errorf(ctx, "Server forced to shutdown:%v", err)
		}
	}
}

func CreateGinServer(contextPath string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(middleware.LoggerMiddleware(), middleware.RecoveryMiddleware(), metric.PrometheusMiddleware())
	ginRouter.GET(contextPath+"/health", func(c *gin.Context) { c.String(http.StatusOK, "UP") })
	ginRouter.GET(contextPath+"/metrics", gin.WrapH(promhttp.Handler()))
	ginRouter.GET(contextPath+"/prometheus", gin.WrapH(promhttp.Handler()))
	return ginRouter
}

// InitWebServer 启动Web服务
func InitWebServer(ctx context.Context, webGroups []config.WebGroup, validatorMap map[string]validator.Func, webMiddlewares ...gin.HandlerFunc) {
	var contextPath = ""
	if config.GlobalConf.WebServer.ContextPath != "" {
		contextPath = config.GlobalConf.WebServer.ContextPath
	}
	// 初始化Gin和HTTP服务
	GinWebRouter = CreateGinServer(contextPath)
	if len(webMiddlewares) > 0 {
		GinWebRouter.Use(webMiddlewares...)
	}
	for _, handler := range webGroups {
		group := GinWebRouter.Group(contextPath + handler.Path)
		for _, webPath := range handler.WebPaths {
			group.Handle(webPath.Method, webPath.Path, webPath.Handler)
		}
	}

	port := fmt.Sprintf(":%s", config.GlobalConf.WebServer.Port)
	httpServer = &http.Server{
		Addr:    port,
		Handler: GinWebRouter,
	}
	util.InitValidator(validatorMap)

	// 启动HTTP服务
	go func() {
		glog.Infof(ctx, "Web服务启动,监听端口:%v", config.GlobalConf.WebServer.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			glog.Error(ctx, "Start web http server failed,errs:"+err.Error())
		}
	}()
}

type GinHandlerFunc struct {
	Method string
	Path   string
	Func   gin.HandlerFunc
}
