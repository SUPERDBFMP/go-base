package go_base

import (
	"context"
	"fmt"
	"go-base/config"
	"go-base/db"
	"go-base/glog"
	"go-base/redis"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Bootstrap(ctx context.Context, configPath string) {
	//serviceWg := &sync.WaitGroup{}
	// 初始化优雅停机信号通道
	//stopChan := make(chan struct{})
	//SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL) // 增加SIGKILL捕获

	config.InitConfig(configPath)
	// Init logger
	glog.InitLogger()
	if config.GlobalConf.Redis != nil {
		redis.InitRedis()
	}
	if config.GlobalConf.MySQL != nil {
		db.InitMysql()
	}

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

	//关闭web

	// 关闭数据库连接（单独设置超时）
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	if err := db.CloseDB(dbCtx); err != nil {
		glog.Errorf(ctx, "数据库关闭失败:%v", err)
	}

	// 关闭Redis连接
	if config.GlobalConf.Redis != nil {
		redisClient := redis.GetRedis(ctx)
		if redisClient != nil {
			redisClient.Close()
			glog.Info(ctx, "Redis连接已关闭")
		}
	}
	glog.Info(ctx, "优雅停机流程完成,程序退出")
	close(done)
	os.Exit(1)
}
