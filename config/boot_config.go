package config

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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

func WithWebValidators(validators map[string]validator.Func) BootOption {
	return func(bc *BootstrapConfig) {
		bc.WebValidators = validators
	}
}

type BootstrapConfig struct {
	WebApi         []WebGroup
	WebMiddlewares []gin.HandlerFunc
	WebValidators  map[string]validator.Func
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
