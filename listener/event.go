package listener

import (
	"time"

	"github.com/SUPERDBFMP/go-base/config"
)

// AppConfigLoadedEvent 启动时加载配置完成
type AppConfigLoadedEvent struct {
	Time            time.Time
	BootstrapConfig *config.BootstrapConfig
}

func (receiver *AppConfigLoadedEvent) SupportAsync() bool {
	return false
}

type AppShutdownEvent struct {
	Time time.Time
}

func (receiver *AppShutdownEvent) SupportAsync() bool {
	return false
}

type AppWebServerStartedEvent struct {
	Time time.Time
}

func (receiver *AppWebServerStartedEvent) SupportAsync() bool {
	return true
}

type AppWebServerStoppedEvent struct {
	Time time.Time
}

func (receiver *AppWebServerStoppedEvent) SupportAsync() bool {
	return true
}
