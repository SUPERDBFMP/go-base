package listener

import (
	"context"
	"go-base/glog"
	"time"
)

// AppConfigLoadedEvent 启动时加载配置完成
type AppConfigLoadedEvent struct {
	Time time.Time
}

func (receiver *AppConfigLoadedEvent) SupportAsync() bool {
	return false
}

type AppConfigLoadedEventListener struct{}

func (ace *AppConfigLoadedEventListener) OnApplicationEvent(ctx context.Context, event *AppConfigLoadedEvent) {
	glog.Infof(ctx, "AppConfigLoadedEvent: %v", event.Time)
}

type AppShutdownEvent struct {
	Time time.Time
}

func (receiver *AppShutdownEvent) SupportAsync() bool {
	return false
}

func (receiver *AppShutdownEvent) OnApplicationEvent(ctx context.Context, event *AppShutdownEvent) {
	glog.Infof(ctx, "AppShutdownEvent: %v", event.Time)
}

type AppWebServerStartedEvent struct {
	Time time.Time
}

func (receiver *AppWebServerStartedEvent) SupportAsync() bool {
	return true
}

type AppWebServerStartedEventListener struct{}

func (ace *AppWebServerStartedEventListener) OnApplicationEvent(ctx context.Context, event *AppWebServerStartedEvent) {
	glog.Infof(ctx, "AppWebServerStartedEvent: %v", event.Time)
}

type AppWebServerStoppedEvent struct {
	Time time.Time
}

func (receiver *AppWebServerStoppedEvent) SupportAsync() bool {
	return true
}

type AppWebServerStoppedEventListener struct{}

func (ace *AppWebServerStoppedEventListener) OnApplicationEvent(ctx context.Context, event *AppWebServerStoppedEvent) {
	glog.Infof(ctx, "AppWebServerStoppedEvent: %v", event.Time)
}
