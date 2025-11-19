package config

import "github.com/gin-gonic/gin"

type BootOption func(*BootstrapConfig)

func WithWebApi(api []WebGroup) BootOption {
	return func(bc *BootstrapConfig) {
		bc.WebApi = api
	}
}

func WithWebMiddlewares(hf []gin.HandlerFunc) BootOption {
	return func(bc *BootstrapConfig) {
		bc.WebMiddlewares = hf
	}
}

type BootstrapConfig struct {
	WebApi         []WebGroup
	WebMiddlewares []gin.HandlerFunc
}

type WebGroup struct {
	Path     string
	WebPaths []WebPath
}

type WebPath struct {
	Path    string
	Method  string
	Handler gin.HandlerFunc
}
