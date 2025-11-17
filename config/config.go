package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v3"
)

const (
	DefaultGroup = "DEFAULT_GROUP"

	LoggerDataId     = "logger"
	MySqlDataId      = "mysql"
	RedisDataId      = "redis"
	OssDataOddId     = "oss"
	PowerJobDataId   = "powerjob"
	GrpcDataId       = "grpc"
	PrometheusDataId = "prometheus"
	WebDataId        = "web"
)

var GlobalConf *GlobalConfig
var NaCosClient config_client.IConfigClient

// NaCosConfig 专门用于NaCos的配置信息
type NaCosConfig struct {
	ServerAddr string `yaml:"server-addr"` // NaCos服务器地址
	UserName   string `yaml:"user-name"`   // NaCos用户名
	Password   string `yaml:"password"`    // NaCos <PASSWORD>
	Namespace  string `yaml:"namespace"`   // NaCos命名空间
}

// LoggerConfig 日志配置结构体
type LoggerConfig struct {
	Level      uint32 `yaml:"level"`       // 日志级别
	Filename   string `yaml:"filename"`    // 日志文件名
	MaxSize    int    `yaml:"max-size"`    // 日志文件最大大小
	MaxAge     int    `yaml:"max-age"`     // 日志文件最大保存时间
	MaxBackups int    `yaml:"max-backups"` // 日志文件最大保存个数
	Compress   bool   `yaml:"compress"`    // 日志文件是否压缩
}

// MySqlConfig mysql配置
type MySqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	UserName string `yaml:"user-name"`
	Password string `yaml:"password"`
	DbName   string `yaml:"db-name"`
	MaxIdle  int    `yaml:"max-idle"` //最大空闲连接
	MaxConn  int    `yaml:"max-conn"` //最大连接数
	MaxLife  int    `yaml:"max-life"` //连接生命周期，单位为分钟
}

// RedisConfig Redis配置结构体
type RedisConfig struct {
	ServerAddress string `yaml:"server-address"` // Redis服务器地址
	Password      string `yaml:"password"`       // Redis密码
	DB            int    `yaml:"db"`             // Redis数据库
	PoolSize      int    `yaml:"pool-size"`      // Redis连接池大小
	MinIdleCones  int    `yaml:"min-idle-cones"` // Redis最小空闲连接数
}

type GlobalConfig struct {
	NaCos  *NaCosConfig  `yaml:"nacos"`
	Logger *LoggerConfig `yaml:"logger"`
	MySQL  *MySqlConfig  `yaml:"mysql"`
	Redis  *RedisConfig  `yaml:"redis"`
}

type Option func(*GlobalConfig)

func WithNaCosConfig(nc *NaCosConfig) Option {
	return func(config *GlobalConfig) {
		config.NaCos = nc
	}
}
func WithLoggerConfig(logger *LoggerConfig) Option {
	return func(config *GlobalConfig) {
		config.Logger = logger
	}
}
func WithMySqlConfig(mysql *MySqlConfig) Option {
	return func(config *GlobalConfig) {
		config.MySQL = mysql
	}
}

func WithRedisConfig(redis *RedisConfig) Option {
	return func(config *GlobalConfig) {
		config.Redis = redis
	}
}

// 初始化配置
func InitConfig(localConfigPath string) {
	if localConfigPath == "" {
		localConfigPath = "./config/config.yml"
	}
	fileByte, err := os.ReadFile(localConfigPath)
	if err != nil {
		_ = fmt.Errorf("read config file path:[%s],err,%v", localConfigPath, err)
		panic(err)
	}
	GlobalConf = new(GlobalConfig)
	err = yaml.Unmarshal(fileByte, GlobalConf)
	if err != nil {
		//logrus.Warnf("Parse yaml config[%s] from Nacos err: %v,use default config", content, err)
	}
	if GlobalConf.NaCos != nil {
		err = initNaCos(GlobalConf.NaCos)
		if err != nil {
			panic(fmt.Sprintf("初始化Nacos客户端错误: %s", err))
		}
		loadLoggerConfig()
		loadMysqlConfig()
		loadRedisConfig()
	}
}

// 初始化NaCos
func initNaCos(config *NaCosConfig) error {
	if config.ServerAddr == "" {
		return errors.New("nacos服务器地址未配置")
	}

	address := strings.Split(config.ServerAddr, ":")
	if port, err := strconv.Atoi(address[1]); err != nil {
		return fmt.Errorf("nacos服务器端口%s配置错误", address[1])
	} else {

		// Nacos服务器配置
		serverConfigs := []constant.ServerConfig{{IpAddr: address[0], Port: uint64(port)}}

		// 客户端配置
		clientConfig := constant.ClientConfig{
			Username:            config.UserName,
			Password:            config.Password,
			NamespaceId:         config.Namespace,
			NotLoadCacheAtStart: true,
		}

		// 创建配置客户端
		NaCosClient, err = clients.NewConfigClient(
			vo.NacosClientParam{
				ClientConfig:  &clientConfig,
				ServerConfigs: serverConfigs,
			},
		)
		return err
	}
}

// 默认配置
var defaultLoggerParam = &LoggerConfig{
	Filename:   "./base.log",
	MaxSize:    50,
	MaxBackups: 30,
	MaxAge:     30,
	Compress:   true,
	Level:      5,
}

// 加载日志配置
func loadLoggerConfig() {
	if GlobalConf.NaCos == nil {
		return
	}
	// 从Nacos获取并解析PowerJob配置
	content, err := NaCosClient.GetConfig(
		vo.ConfigParam{
			DataId: LoggerDataId,
			Group:  DefaultGroup,
		},
	)
	if err != nil {
		//logrus.Warnf("Fetch config from Nacos with data id[%s] err:%v,use default config", LoggerDataId, err)
		WithLoggerConfig(defaultLoggerParam)
		return
	}
	if content == "" {
		//logrus.Warnf("Fetch config from Nacos with data id[%s] is blank,use default config", LoggerDataId)
		WithLoggerConfig(defaultLoggerParam)
		return
	}
	var config LoggerConfig
	if err = yaml.Unmarshal([]byte(content), &config); err != nil {
		//logrus.Warnf("Parse yaml config[%s] from Nacos err: %v,use default config", content, err)
		WithLoggerConfig(&config)
	}
}

// 加载MySQL配置
func loadMysqlConfig() {
	content, err := NaCosClient.GetConfig(vo.ConfigParam{DataId: MySqlDataId, Group: DefaultGroup})
	if err != nil {
		panic(fmt.Sprintf("Fetch config from Nacos with data id[%s]err:%s", MySqlDataId, err))
	}
	if content == "" {
		//logrus.Warnf("Fetch config from Nacos with data id[%s] is blank", MySqlDataId)
		return
	}

	var config MySqlConfig
	if err = yaml.Unmarshal([]byte(content), &config); err != nil {
		panic(fmt.Sprintf("Parse yaml config[%s] from Nacos err: %v", content, err))
	}
	WithMySqlConfig(&config)
}

// 加载Redis配置
func loadRedisConfig() {
	content, err := NaCosClient.GetConfig(vo.ConfigParam{DataId: RedisDataId, Group: DefaultGroup})
	if err != nil {
		panic(fmt.Sprintf("Fetch config from Nacos with data id[%s]err:%s", RedisDataId, err))
	}
	if content == "" {
		//logrus.Warnf("Fetch config from Nacos with data id[%s] is blank", MySqlDataId)
		return
	}

	var config RedisConfig
	if err = yaml.Unmarshal([]byte(content), &config); err != nil {
		panic(fmt.Sprintf("Parse yaml config[%s] from Nacos err: %v", content, err))
	}
	WithRedisConfig(&config)
}

// ChangeHandler 配置变更处理器函数
type ChangeHandler func(data string)

// RegisterConfigChangeHandler 注册配置变更监听
func RegisterConfigChangeHandler(dataId, group string, handler ChangeHandler) {
	if err := NaCosClient.ListenConfig(
		vo.ConfigParam{
			DataId:   dataId,
			Group:    group,
			OnChange: func(namespace, group, dataId, data string) { handler(data) },
		},
	); err != nil {
		panic("register nacos config change listener failed, error: " + err.Error())
	}
}
