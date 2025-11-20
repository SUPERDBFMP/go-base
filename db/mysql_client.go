package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/SUPERDBFMP/go-base/config"
	"github.com/SUPERDBFMP/go-base/glog"
	"github.com/SUPERDBFMP/go-base/listener"
	"github.com/SUPERDBFMP/go-base/util"

	"github.com/acmestack/gorm-plus/gplus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// GlobalDB 全局数据库实例
var GlobalDB *gorm.DB

func init() {
	listener.AddTypedApplicationListener(&AppConfigLoadedEventListener{})
	listener.AddTypedApplicationListener(&AppShutDownEventListener{})
}

// InitMysql 初始化数据库
func InitMysql(ctx context.Context) {
	mysqlConfig := config.GlobalConf.MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlConfig.UserName, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.DbName,
	)

	gormLogger := newGormLogger()
	gormLogger.SlowThreshold = 3000 * time.Millisecond

	db, err := gorm.Open(
		mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		},
	)
	if err != nil {
		panic(fmt.Sprintf("连接数据库[%s]失败,异常:%v", mysqlConfig.DbName, err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sqlDB: " + err.Error())
	}
	if err = sqlDB.Ping(); err != nil {
		panic("failed to ping database: " + err.Error())
	}
	// 配置连接池
	sqlDB.SetMaxIdleConns(mysqlConfig.MaxIdle)                                 // 最大空闲连接数
	sqlDB.SetMaxOpenConns(mysqlConfig.MaxConn)                                 // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Duration(mysqlConfig.MaxLife) * time.Minute) // 连接最大生命周期
	GlobalDB = db
	if err := setupGlobalIDHook(db); err != nil {
		panic("failed to setup global ID hook: " + err.Error())
	}
	gplus.Init(db)
	glog.Infof(ctx, "Mysql connected successfully!")
}

// setupGlobalIDHook 注册全局ID生成钩子
func setupGlobalIDHook(db *gorm.DB) error {
	err := db.Callback().Create().Before("gorm:create").Register(
		"global_gen_id", func(d *gorm.DB) {
			if d.Statement.Schema == nil || len(d.Statement.Schema.PrimaryFields) == 0 {
				return
			}
			idField := d.Statement.Schema.PrimaryFields[0]

			// 获取待创建的记录（可能是单条或切片）
			reflectValue := d.Statement.ReflectValue
			if reflectValue.Kind() == reflect.Ptr {
				reflectValue = reflectValue.Elem() // 解引用指针
			}

			// 处理批量创建（切片类型）
			if reflectValue.Kind() == reflect.Slice {
				for i := 0; i < reflectValue.Len(); i++ {
					elem := reflectValue.Index(i)
					if err := setIDForElement(elem, idField); err != nil {
						_ = d.AddError(err)
						return
					}
				}
			} else {
				// 处理单条创建（非切片类型）
				if err := setIDForElement(reflectValue, idField); err != nil {
					_ = d.AddError(err)
					return
				}
			}
		},
	)
	if err != nil {
		return fmt.Errorf("注册全局ID生成钩子失败: %w", err)
	}
	return nil
}

// 为单个元素设置ID（通过反射直接操作,不依赖ValueOf）
// elem是model对象
func setIDForElement(elem reflect.Value, idField *schema.Field) error {
	// 确保元素是结构体（如果是指针则解引用）
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("元素不是结构体类型,无法设置ID")
	}

	// 通过字段名获取ID字段的反射值
	idFieldValue := elem.FieldByName(idField.Name)
	if !idFieldValue.IsValid() {
		return fmt.Errorf("结构体中不存在ID字段: %s", idField.Name)
	}
	if !idFieldValue.CanSet() {
		return fmt.Errorf("ID字段不可设置,可能是未导出字段")
	}

	// 检查ID是否为零值
	if idFieldValue.IsZero() {
		// 生成新ID并设置
		newID := util.GenerateBigintID()
		// 确保类型匹配（int64）
		if idFieldValue.Kind() != reflect.Int64 {
			return fmt.Errorf("ID字段类型不是int64,无法设置")
		}
		idFieldValue.SetInt(newID)
		//idFieldValue.Set(reflect.Zero(idFieldValue.Type()))
	} else {
		// 验证已有ID的类型
		if idFieldValue.Kind() != reflect.Int64 {
			return fmt.Errorf("ID字段类型错误,预期int64,实际%v", idFieldValue.Kind())
		}
	}

	return nil
}

// GormLogger 实现 GORM 的 logger.Interface 接口
type GormLogger struct {
	LogLevel      logger.LogLevel // GORM 日志级别
	SlowThreshold time.Duration   // 慢查询阈值
}

// newGormLogger 创建一个新的 GORM-Logrus 适配器
func newGormLogger() *GormLogger {
	return &GormLogger{
		SlowThreshold: 200 * time.Millisecond, // 默认慢查询阈值
	}
}

// LogMode 设置当前的logger level
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		glog.Infof(ctx, msg, data...)
	}
}
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		glog.Infof(ctx, msg, data...)
	}
}
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		glog.Infof(ctx, msg, data...)
	}
}
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	// 计算执行时间
	elapsed := time.Since(begin)
	// 获取 SQL 语句和影响行数
	sql, rows := fc()

	// 处理错误
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		sql = strings.ReplaceAll(sql, "\n", "")
		sql = strings.ReplaceAll(sql, "\t", "")
		glog.Errorf(ctx, "SQL 执行错误 %s", sql)
		return
	}

	// 慢查询警告
	var slow = ""
	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		//entry.Warn("慢查询警告")
		slow = fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
	}

	// 正常 SQL 日志
	if l.LogLevel >= logger.Info {
		sql = strings.ReplaceAll(sql, "\n", "")
		sql = strings.ReplaceAll(sql, "\t", "")
		glog.Infof(ctx, "%s | rows:%v | elapsed:%dms %s", sql, rows, elapsed.Milliseconds(), slow)
	}
}

type AppConfigLoadedEventListener struct{}

func (ace *AppConfigLoadedEventListener) GetOrder() int {
	return 0
}

func (ace *AppConfigLoadedEventListener) OnApplicationEvent(ctx context.Context, event *listener.AppConfigLoadedEvent) {
	glog.Infof(ctx, "AppConfigLoadedEvent: %v", event.Time)
	if config.GlobalConf.MySQL != nil {
		InitMysql(ctx)
	}
}

type AppShutDownEventListener struct{}

func (l *AppShutDownEventListener) GetOrder() int {
	return 2
}

func (l *AppShutDownEventListener) OnApplicationEvent(ctx context.Context, event *listener.AppShutdownEvent) {
	// 关闭数据库连接（单独设置超时）
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	if err := CloseDB(dbCtx); err != nil {
		glog.Errorf(ctx, "数据库关闭失败:%v", err)
	}
}

// CloseDB 关闭GORM数据库连接（带上下文超时）
func CloseDB(ctx context.Context) error {
	glog.Info(ctx, "开始关闭数据库连接...")
	if GlobalDB == nil {
		glog.Info(ctx, "数据库连接未初始化,无需关闭")
		return nil
	}

	sqlDB, err := GlobalDB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败:%w", err)
	}

	// 使用带超时的上下文关闭连接
	// 注意:sql.DB.Close()不支持直接传入ctx,这里用通道+超时模拟
	closeChan := make(chan error, 1)
	go func() {
		closeChan <- sqlDB.Close()
	}()

	select {
	case err := <-closeChan:
		if err != nil {
			return fmt.Errorf("数据库连接关闭失败:%w", err)
		}
		glog.Info(ctx, "数据库连接已优雅关闭")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("数据库关闭超时:%w", ctx.Err())
	}
}
