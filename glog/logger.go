package glog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-base/config"
	"go-base/trace"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

// 全局互斥锁
var logMutex sync.Mutex

// 全局日志轮转器（用于配置变更时关闭旧实例）
var globalRotator *DailySizeRotator

var appName string

var podName string

var fieldOrder = []string{"path", "method", "ip", "status", "cost"}

var fieldOrderMap = map[string]bool{
	"path":   true,
	"method": true,
	"ip":     true,
	"status": true,
	"cost":   true,
}

// DailySizeRotator 结合每日切换和大小切割的日志轮转器
type DailySizeRotator struct {
	mu          sync.Mutex          // 并发安全锁
	currentDate string              // 当前日期（YYYY-MM-DD）
	lumberjack  *lumberjack.Logger  // 当前lumberjack实例
	config      config.LoggerConfig // 日志配置
	ticker      *time.Ticker        // 每日切换定时器
}

// NewDailySizeRotator 创建轮转器实例
func NewDailySizeRotator(config config.LoggerConfig) (*DailySizeRotator, error) {
	rotator := &DailySizeRotator{
		config: config,
	}
	// 初始化当天日志文件
	if err := rotator.resetLumberjack(); err != nil {
		return nil, err
	}
	// 启动每日切换定时器（每天0点触发）
	rotator.startDailyTicker()
	return rotator, nil
}

// 生成当天的日志文件名（如 "logs/app-2025-10-21.log"）
func (d *DailySizeRotator) todayFilename() string {
	podName := getEnv("POD_NAME", "default")
	replace := fmt.Sprintf(d.config.Filename, podName)
	today := time.Now().Format("2006-01-02")
	// 提取原文件名的前缀和后缀（如 "logs/app.log" → 前缀"logs/app", 后缀".log"）
	ext := filepath.Ext(replace)
	prefix := strings.TrimSuffix(replace, ext)
	return fmt.Sprintf("%s-%s%s", prefix, today, ext)
}

func (d *DailySizeRotator) filename() string {
	podName := getEnv("POD_NAME", "default")
	if strings.Contains(d.config.Filename, "${POD_NAME}") {
		return strings.Replace(d.config.Filename, "${POD_NAME}", podName, -1)
	}
	return d.config.Filename
}

// 重置lumberjack实例（切换到当天文件）
func (d *DailySizeRotator) resetLumberjack() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	// 日期未变则无需重置
	if today == d.currentDate && d.lumberjack != nil {
		return nil
	}

	// 关闭旧实例（确保缓冲刷新）
	if d.lumberjack != nil {
		if err := d.lumberjack.Close(); err != nil {
			logrus.Warnf("关闭旧日志文件失败: %v", err)
		}
	}

	// 创建新的lumberjack实例（当天文件）
	filename := d.filename()
	d.lumberjack = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    d.config.MaxSize,    // 当天内按大小切割（MB）
		MaxBackups: d.config.MaxBackups, // 最大备份文件数
		MaxAge:     d.config.MaxAge,     // 日志保留天数
		Compress:   d.config.Compress,   // 压缩旧文件
		LocalTime:  true,
	}
	d.currentDate = today
	return nil
}

// 启动每日0点切换定时器
func (d *DailySizeRotator) startDailyTicker() {
	// 计算距离下次0点的时间
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	duration := nextMidnight.Sub(now)

	// 首次延迟后触发,之后每天触发一次
	d.ticker = time.NewTicker(duration)
	go func() {
		for {
			<-d.ticker.C
			// 切换到当天文件
			if err := d.resetLumberjack(); err != nil {
				logrus.Errorf("每日日志切换失败: %v", err)
			}
			// 重置定时器为24小时
			d.ticker.Reset(24 * time.Hour)
		}
	}()
}

// Write 实现io.Writer接口,将日志写入当前lumberjack实例
func (d *DailySizeRotator) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lumberjack == nil {
		return 0, os.ErrInvalid
	}
	return d.lumberjack.Write(p)
}

// Close 关闭轮转器（释放资源）
func (d *DailySizeRotator) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ticker != nil {
		d.ticker.Stop()
	}
	if d.lumberjack != nil {
		return d.lumberjack.Close()
	}
	return nil
}

// InitLogger 初始化日志配置（支持每日+大小轮转）
func InitLogger(ctx context.Context) {
	logMutex.Lock()
	defer logMutex.Unlock() // 确保释放锁
	params := config.GlobalConf.Logger

	// 读取环境变量
	appName = getEnv("APP_NAME", "")
	podName = getEnv("POD_NAME", "")

	// 关闭旧的轮转器（配置变更时）
	if globalRotator != nil {
		if err := globalRotator.Close(); err != nil {
			logrus.Warnf("关闭旧日志轮转器失败: %v", err)
		}
		globalRotator = nil
	}

	// 初始化每日+大小轮转的日志writer
	fileRotator, err := NewDailySizeRotator(*params)
	if err != nil {
		logrus.Fatalf("初始化日志轮转器失败: %v", err)
	}
	globalRotator = fileRotator

	// 控制台使用自定义格式（|分隔）
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&SimpleTextFormatter{})

	// 添加文件Hook，使用JSON格式
	logrus.AddHook(
		&FileHook{
			writer: fileRotator,
			formatter: &OrderedJSONFormatter{
				FieldOrder:      []string{"@timestamp", "level", "traceId", "caller", "message"},
				TimestampFormat: "2006-01-02 15:04:05.000",
			},
		},
	)

	// 设置日志级别
	level := logrus.Level(params.Level)
	if level < logrus.PanicLevel || level > logrus.TraceLevel {
		logrus.Warnf("无效的日志级别: %d,使用默认级别 InfoLevel", params.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	Info(ctx, "init logrus success（支持每日+大小轮转）")

	if config.GlobalConf.NaCos != nil {
		// 注册配置变更处理器
		config.RegisterConfigChangeHandler(
			config.LoggerDataId, config.DefaultGroup, func(data string) {
				Infof(
					ctx, "DataId:%s,Group:%s 配置发生变更为:%s", config.LoggerDataId, config.DefaultGroup, data,
				)
				if err := yaml.Unmarshal([]byte(data), &config.GlobalConf.Logger); err != nil {
					logrus.Warnf("Parse yaml config[%s] from Nacos err: %v,ignore", data, err)
					return
				}
				InitLogger(ctx) // 重新初始化日志
			},
		)
	}
}

// SimpleTextFormatter 简单文本格式化器（|分隔）
type SimpleTextFormatter struct{}

// Format 实现logrus.Formatter接口
func (f *SimpleTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
	level := strings.ToUpper(entry.Level.String())
	message := entry.Message
	// 获取traceId
	traceId := ""
	if tid, ok := entry.Data[trace.TraceIdKey].(string); ok {
		traceId = tid
	}

	// 获取caller信息
	caller := getCaller(8)

	// 格式: timestamp|level|traceId|caller|message
	var parts []string
	parts = append(parts, timestamp, level)
	if traceId != "" {
		parts = append(parts, traceId)
	} else {
		parts = append(parts, "-")
	}
	if caller != "" {
		parts = append(parts, caller)
	} else {
		parts = append(parts, "-")
	}
	parts = append(parts, message)

	// 按预定顺序添加字段
	for _, key := range fieldOrder {
		if value, exists := entry.Data[key]; exists {
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	}
	//for _, key := range fieldOrder {
	//	delete(entry.Data,key )
	//}

	for k, v := range entry.Data {
		if k != trace.TraceIdKey {
			_, ok := fieldOrderMap[k]
			if !ok {
				parts = append(parts, fmt.Sprintf("%v", v))
			}
		}
	}

	return []byte(strings.Join(parts, "|") + "\n"), nil
}

// FileHook 文件输出Hook
type FileHook struct {
	writer    io.Writer
	formatter logrus.Formatter
}

// Levels 返回支持的日志级别
func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 执行Hook
func (hook *FileHook) Fire(entry *logrus.Entry) error {
	data, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = hook.writer.Write(data)
	return err
}

// OrderedJSONFormatter 自定义JSON格式化器
type OrderedJSONFormatter struct {
	FieldOrder      []string
	TimestampFormat string
}

// Format 实现logrus.Format接口
func (f *OrderedJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+4)

	// 设置固定字段
	if f.TimestampFormat == "" {
		f.TimestampFormat = "2006-01-02 15:04:05.000"
	}
	data["@timestamp"] = entry.Time.Format(f.TimestampFormat)
	data["level"] = strings.ToUpper(entry.Level.String())
	data["message"] = entry.Message
	if appName != "" {
		data["appName"] = appName
	}
	if podName != "" {
		data["podName"] = podName
	}

	if entry.Caller != nil {
		caller := getCaller(10)
		data["caller"] = caller
		if idx := strings.Index(caller, ":"); idx != -1 {
			data["logger_name"] = caller[:idx]
		}
	}

	// 复制额外字段
	for k, v := range entry.Data {
		data[k] = v
	}

	// 构建有序JSON
	keys := make([]string, 0, len(data))
	keySet := make(map[string]struct{}, len(data))
	for _, field := range f.FieldOrder {
		if _, exists := data[field]; exists {
			keys = append(keys, field)
			keySet[field] = struct{}{}
		}
	}
	var otherKeys []string
	for k := range data {
		if _, exists := keySet[k]; !exists {
			otherKeys = append(otherKeys, k)
		}
	}
	sort.Strings(otherKeys)
	keys = append(keys, otherKeys...)

	buf := make([]byte, 0, 256)
	buf = append(buf, '{')
	for i, key := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyBytes, _ := marshalDisableHtml(key)
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		valBytes, _ := marshalDisableHtml(data[key])
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}', '\n')

	return buf, nil
}

// getCaller 获取真实调用者信息
func getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		// 如果获取失败，逐步减小skip重试
		for i := skip - 1; i >= 3; i-- {
			pc, file, line, ok = runtime.Caller(i)
			if ok {
				break
			}
		}
	}
	if !ok {
		return "-"
	}

	// 获取函数名
	funcName := runtime.FuncForPC(pc).Name()
	if lastSlash := strings.LastIndex(funcName, "/"); lastSlash != -1 {
		funcName = funcName[lastSlash+1:]
	}
	if dotIndex := strings.Index(funcName, "."); dotIndex != -1 {
		funcName = funcName[dotIndex+1:]
	}

	return fmt.Sprintf("%s:%s:%d", filepath.Base(file), funcName, line)
}

func marshalDisableHtml(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // 禁用转义
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// 移除Encode自动添加的换行符（与Marshal保持一致）
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		logrus.Warnf("No found env %s value,use default value %s", key, fallback)
		return fallback
	}
	return value
}

func Debug(ctx context.Context, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Debug(msg)
}

func Info(ctx context.Context, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Info(msg)
}

func Warn(ctx context.Context, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Warn(msg)
}

func Error(ctx context.Context, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Error(msg)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Debugf(format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Infof(format, args...)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Warnf(format, args...)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).Errorf(format, args...)
}

func DebugWithFields(ctx context.Context, fields logrus.Fields, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Debug(msg)
}

func InfoWithFields(ctx context.Context, fields logrus.Fields, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Info(msg)
}

func WarnWithFields(ctx context.Context, fields logrus.Fields, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Warn(msg)
}

func ErrorWithFields(ctx context.Context, fields logrus.Fields, msg string) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Error(msg)
}

func DebugfWithFields(ctx context.Context, fields logrus.Fields, msg string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Debugf(msg, args...)
}

func InfofWithFields(ctx context.Context, fields logrus.Fields, msg string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Infof(msg, args...)
}

func WarnfWithFields(ctx context.Context, fields logrus.Fields, msg string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Warnf(msg, args...)
}

func ErrorfWithFields(ctx context.Context, fields logrus.Fields, msg string, args ...interface{}) {
	logrus.WithContext(ctx).WithFields(trace.BuildTraceField(ctx)).WithFields(fields).Errorf(msg, args...)
}
