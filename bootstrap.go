package go_base

import (
	"context"
	"fmt"
	"go-base/config"
	"go-base/glog"
	"go-base/listener"
	"go-base/trace"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Bootstrap(ctx context.Context, configPath string, options ...config.BootOption) {
	ctx = context.WithValue(ctx, trace.TraceIdKey, "main")
	bootstrapConfig := &config.BootstrapConfig{}
	// 初始化优雅停机信号通道
	//stopChan := make(chan struct{})
	//SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL) // 增加SIGKILL捕获
	config.InitConfig(configPath)
	// Init logger
	glog.InitLogger(ctx)
	for _, option := range options {
		option(bootstrapConfig)
	}
	listener.PublishApplicationEvent(ctx, &listener.AppConfigLoadedEvent{
		Time:            time.Now(),
		BootstrapConfig: bootstrapConfig,
	})
	//做一些钩子
	//等待接收退出信号
	sig := <-sigChan
	glog.Infof(ctx, "收到退出信号:%v,开始优雅停机...", sig)
	// 关闭信号广播（通知所有服务开始关闭）
	//close(stopChan)
	// 等待所有服务关闭（设置最长等待时间30秒）
	shutdownTimeout := 30 * time.Second
	done := make(chan struct{})
	go func() {
		//serviceWg.Wait() // 等待所有服务关闭
		select {
		case <-done:
			glog.Infof(ctx, "所有服务已正常关闭")
		case <-time.After(shutdownTimeout):
			glog.Warn(ctx, fmt.Sprintf("服务关闭超时（%v）,可能存在未完成的任务", shutdownTimeout))
			os.Exit(1)
		}
	}()
	glog.Info(ctx, "开始优雅停机")
	listener.PublishApplicationEvent(ctx, &listener.AppShutdownEvent{
		Time: time.Now(),
	})
	//todo 关闭web,在web中处理
	listener.PublishApplicationEvent(ctx, &listener.AppWebServerStoppedEvent{
		Time: time.Now(),
	})
	glog.Info(ctx, "优雅停机流程完成,程序退出")
	close(done)
	os.Exit(1)
}
