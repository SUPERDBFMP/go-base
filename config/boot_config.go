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

func WithCustomerConfigs(name string, configs interface{}) BootOption {
	return func(bc *BootstrapConfig) {
		if bc.CustomerConfigs == nil {
			bc.CustomerConfigs = make(map[string]interface{})
			bc.CustomerConfigs[name] = configs
		} else {
			bc.CustomerConfigs[name] = configs
		}
	}
}

type BootstrapConfig struct {
	WebApi          []WebGroup
	WebMiddlewares  []gin.HandlerFunc
	WebValidators   map[string]validator.Func
	CustomerConfigs map[string]interface{}
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
