package redis

import (
	"context"
	"errors"
	"fmt"
	"go-base/config"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Client Redis客户端结构体
type Client struct {
	client *redis.Client
	ctx    context.Context
}

var originRedisClient *redis.Client

var maxRetryCount = 3

// InitRedis 创建新的Redis客户端
func InitRedis() {
	redisConf := config.GlobalConf.Redis
	if redisConf == nil {
		return
	}
	if redisConf.ServerAddress == "" {
		panic("No found redis config address from nacos")
	}
	if redisConf.PoolSize == 0 {
		redisConf.PoolSize = 10
	}
	if redisConf.MinIdleCones == 0 {
		redisConf.MinIdleCones = 5
	}

	client := redis.NewClient(
		&redis.Options{
			Addr:         redisConf.ServerAddress,
			Password:     redisConf.Password,
			DB:           redisConf.DB,
			PoolSize:     redisConf.PoolSize,
			MinIdleConns: redisConf.MinIdleCones,
		},
	)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		//rlog.Errorf(ctx, "无法连接到Redis: %v", err)
		panic("Can't connection redis server")
	}

	originRedisClient = client
	//rlog.Infof(ctx, "Redis connected successfully.")
}

// GetRedis 获取Redis客户端
func GetRedis(ctx context.Context) *Client {
	if originRedisClient == nil {
		panic("Please init redis client")
	}
	return &Client{
		client: originRedisClient,
		ctx:    ctx,
	}
}

// Set 设置键值对
func (r *Client) Set(key, value string, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

// SetNX 设置键值对，如果键不存在则设置成功
func (r *Client) SetNX(key, value string, expiration time.Duration) (bool, error) {
	// 使用 SetNX 命令设置键值对，如果键不存在则设置成功，否则设置失败
	result, err := r.client.SetNX(r.ctx, key, value, expiration).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// Get 获取键的值
func (r *Client) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// Del 删除键
func (r *Client) Del(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// HSet 设置哈希表字段
func (r *Client) HSet(key, field, value string) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

// HGet 获取哈希表字段值
func (r *Client) HGet(key, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

// LPush 向列表左侧添加元素
func (r *Client) LPush(key, value string) error {
	return r.client.LPush(r.ctx, key, value).Err()
}

// LRange 获取列表指定范围内的元素
func (r *Client) LRange(key string, start, end int64) ([]string, error) {
	return r.client.LRange(r.ctx, key, start, end).Result()
}

// Close 关闭Redis连接
func (r *Client) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// DistributedLock 分布式锁结构体
type DistributedLock struct {
	rdb        *redis.Client // Redis客户端
	key        string        // 锁的key
	value      string        // 锁的唯一标识（UUID）
	expiration time.Duration // 锁的过期时间
	ticker     *time.Ticker  // 续期定时器
	stopChan   chan struct{} // 停止续期的信号
	isLocked   bool          // 是否持有锁
}

// NewDistributedLock 创建分布式锁实例
func NewDistributedLock(key string, expiration time.Duration) *DistributedLock {
	if originRedisClient == nil {
		InitRedis()
		if originRedisClient == nil {
			panic("Please init redis client")
		}
	}
	return &DistributedLock{
		rdb:        originRedisClient,
		key:        key,
		value:      uuid.New().String(), // 生成唯一UUID作为value
		expiration: expiration,
		stopChan:   make(chan struct{}),
	}
}

// Lock 获取分布式锁
func (l *DistributedLock) Lock(ctx context.Context) (bool, error) {
	retryCount := 0
	// 使用 SetNX 方法:等价于 Redis 命令 "SET key value NX PX <expiration>"
	// 第4个参数 expiration 直接指定过期时间（毫秒级）
	boolCmd := l.rdb.SetNX(ctx, l.key, l.value, l.expiration)

	// 获取结果（是否成功获取锁）
	acquired, err := boolCmd.Result()
	if err != nil {
		//return false, fmt.Errorf("获取锁失败: %w", err)
		retryCount++
	}

	if acquired {
		l.isLocked = true
		l.startRenewal() // 启动自动续期
		return true, nil
	}

	// 锁已被持有
	//自旋等待
	l.ticker = time.NewTicker(100 * time.Millisecond)
	defer l.ticker.Stop()
	for {
		select {
		case <-l.ticker.C:
			boolCmd = l.rdb.SetNX(ctx, l.key, l.value, l.expiration)
			// 获取结果（是否成功获取锁）
			acquired, err = boolCmd.Result()
			if err != nil {
				//todo
				retryCount++
				if retryCount >= maxRetryCount {
					return false, fmt.Errorf("执行redis命令出错,获取锁失败: %w", err)
				}
			}
			if acquired {
				l.isLocked = true
				l.startRenewal()
				return true, nil
			}
		}
	}
}

// TryLock 获取分布式锁
func (l *DistributedLock) TryLock(ctx context.Context) (bool, error) {
	// 使用 SetNX 方法:等价于 Redis 命令 "SET key value NX PX <expiration>"
	// 第4个参数 expiration 直接指定过期时间（毫秒级）
	boolCmd := l.rdb.SetNX(ctx, l.key, l.value, l.expiration)

	// 获取结果（是否成功获取锁）
	acquired, err := boolCmd.Result()
	if err != nil {
		return false, fmt.Errorf("获取锁失败: %w", err)
	}

	if acquired {
		l.isLocked = true
		l.startRenewal() // 启动自动续期
		return true, nil
	}

	// 锁已被持有
	return false, nil
}

// Unlock 释放分布式锁
// ctx: 上下文
// 返回:错误信息
func (l *DistributedLock) Unlock(ctx context.Context) error {
	if !l.isLocked {
		return errors.New("未持有锁,无需释放")
	}

	// Lua脚本:原子性检查并删除锁（仅当value匹配时）
	script := `
		if redis.call('get', KEYS[1]) == ARGV[1] then
			return redis.call('del', KEYS[1])
		else
			return 0
		end
	`
	// 执行脚本:KEYS[1]是锁的key,ARGV[1]是当前锁的value
	result, err := l.rdb.Eval(ctx, script, []string{l.key}, l.value).Int64()
	if err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	// 停止自动续期
	l.stopRenewal()
	l.isLocked = false

	if result == 0 {
		return errors.New("锁已被其他客户端持有或已过期")
	}
	return nil
}

// startRenewal 启动自动续期（看门狗）
func (l *DistributedLock) startRenewal() {
	// 续期间隔:取过期时间的1/3（确保在锁过期前完成续期）
	renewalInterval := l.expiration / 3
	l.ticker = time.NewTicker(renewalInterval)

	go func() {
		for {
			select {
			case <-l.ticker.C:
				// 定时续期
				if err := l.renew(context.Background()); err != nil {
					//rlog.Errorf(context.Background(), "自动续期失败: %v", err)
					// 续期失败时可触发业务降级逻辑（如中断任务）
				}
			case <-l.stopChan:
				// 收到停止信号,退出续期
				return
			}
		}
	}()
}

// stopRenewal 停止自动续期
func (l *DistributedLock) stopRenewal() {
	if l.ticker != nil {
		l.ticker.Stop()
	}
	close(l.stopChan)
}

// renew 延长锁的有效期（原子操作）
func (l *DistributedLock) renew(ctx context.Context) error {
	// Lua脚本:仅当锁的value匹配时,延长过期时间
	script := `
		if redis.call('get', KEYS[1]) == ARGV[1] then
			return redis.call('pexpire', KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	// 执行脚本:ARGV[2]是过期时间（毫秒）
	result, err := l.rdb.Eval(ctx, script, []string{l.key}, l.value, l.expiration.Milliseconds()).Int64()
	if err != nil {
		return fmt.Errorf("续期脚本执行失败: %w", err)
	}
	if result == 0 {
		return errors.New("续期失败,锁已被释放或不属于当前客户端")
	}
	return nil
}
