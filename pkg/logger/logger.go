package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig 日志配置结构体
type LogConfig struct {
	// 基础配置
	AppName   string
	LogLevel  string
	LogFormat string // "text" 或 "json"

	// 控制台输出
	EnableConsole bool
	ConsoleColor  bool

	// 文件输出
	EnableFile bool
	FilePath   string
	FileName   string
	MaxSize    int  // MB
	MaxBackups int  // 最大备份数
	MaxAge     int  // 保留天数
	Compress   bool // 是否压缩备份

	// 统计配置
	EnableStats bool
}

// Stats 线程安全的统计信息收集器
type Stats struct {
	mu           sync.RWMutex
	OverallStats map[string]interface{}
	startTime    time.Time
}

// NewStats 创建统计器
func NewStats() *Stats {
	return &Stats{
		OverallStats: make(map[string]interface{}),
		startTime:    time.Now(),
	}
}

// AddInt 添加整数统计
func (s *Stats) AddInt(key string, value int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.OverallStats[key].(int); ok {
		s.OverallStats[key] = current + value
	} else {
		s.OverallStats[key] = value
	}
}

// AddString 添加字符串统计
func (s *Stats) AddString(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OverallStats[key] = value
}

// Increment 递增计数器
func (s *Stats) Increment(key string) {
	s.AddInt(key, 1)
}

// Set 设置任意类型的值
func (s *Stats) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OverallStats[key] = value
}

// Get 获取统计值
func (s *Stats) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.OverallStats[key]
	return val, ok
}

// GetInt 获取整数统计值
func (s *Stats) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// Clear 清空统计
func (s *Stats) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OverallStats = make(map[string]interface{})
	s.startTime = time.Now()
}

// GetUptime 获取运行时间
func (s *Stats) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.startTime)
}

// OutTableInfo 以表格形式输出统计信息
func (s *Stats) OutTableInfo(writer io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := tablewriter.NewWriter(writer)
	table.Header([]string{"统计项目", "信息"})

	// 添加运行时间
	uptime := s.GetUptime()
	table.Append([]string{"运行时间", fmt.Sprintf("%v", uptime.Round(time.Second))})

	// 添加其他统计项
	for key, value := range s.OverallStats {
		var valueStr string

		switch v := value.(type) {
		case string:
			valueStr = v
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, complex64, complex128:
			valueStr = fmt.Sprintf("%v", v)
		case time.Time:
			valueStr = v.Format("2006-01-02 15:04:05")
		case time.Duration:
			valueStr = v.String()
		case fmt.Stringer:
			valueStr = v.String()
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		table.Append([]string{key, valueStr})
	}

	return table.Render()
}

// Logger 增强的日志记录器
type Logger struct {
	name     string
	mu       sync.RWMutex
	logger   *logrus.Logger
	fileHook *lumberjack.Logger
	config   *LogConfig
	Stats    *Stats
	fields   map[string]interface{}
}

// NewLogger 创建新的日志记录器
func NewLogger(config *LogConfig) (*Logger, error) {
	fmt.Println(config)
	filePath := "./logs/" + config.AppName + "_" + time.Now().Format("2006_01_02_15_04_05")
	if config == nil {
		config = &LogConfig{
			AppName:       "app",
			LogLevel:      "info",
			LogFormat:     "text",
			EnableConsole: true,
			ConsoleColor:  true,
			EnableFile:    false,
			EnableStats:   true,
			MaxSize:       100, // MB
			MaxBackups:    10,
			MaxAge:        30, // days
			Compress:      true,
		}
	}
	config.FilePath = filePath

	logger := logrus.New()

	// 设置日志级别
	level := ParseLogLevel(config.LogLevel)
	logger.SetLevel(level)

	// 设置日志格式
	if strings.ToLower(config.LogFormat) == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     config.ConsoleColor,
			PadLevelText:    true,
		})
	}

	// 设置输出
	var writers []io.Writer

	if config.EnableConsole {
		writers = append(writers, os.Stdout)
	}

	l := &Logger{
		name:   config.AppName,
		logger: logger,
		config: config,
		fields: make(map[string]interface{}),
	}

	// 添加统计功能
	if config.EnableStats {
		l.Stats = NewStats()
		l.Stats.Set("app_name", config.AppName)
		l.Stats.Set("log_level", config.LogLevel)
	}

	// 初始化文件输出
	if config.EnableFile && config.FilePath != "" {
		if err := l.initFileOutput(config); err != nil {
			return nil, fmt.Errorf("初始化文件输出失败: %w", err)
		}
	}

	// 如果有多个输出，使用 MultiWriter
	if len(writers) > 0 {
		if l.fileHook != nil {
			writers = append(writers, l.fileHook)
		}
		logger.SetOutput(io.MultiWriter(writers...))
	} else if l.fileHook != nil {
		// 只有文件输出
		logger.SetOutput(l.fileHook)
	}

	return l, nil
}

// initFileOutput 初始化文件输出
func (l *Logger) initFileOutput(config *LogConfig) error {
	// 确保目录存在
	if err := os.MkdirAll(config.FilePath, 0755); err != nil {
		return err
	}

	// 设置默认文件名
	fileName := config.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("%s.log", config.AppName)
	}

	filePath := filepath.Join(config.FilePath, fileName)

	l.fileHook = &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    config.MaxSize,    // MB
		MaxBackups: config.MaxBackups, // 保留的旧文件数量
		MaxAge:     config.MaxAge,     // 保留天数
		Compress:   config.Compress,   // 是否压缩旧文件
		LocalTime:  true,
	}

	// 记录文件路径到统计
	if l.Stats != nil {
		l.Stats.Set("log_file", filePath)
	}

	return nil
}

// ParseLogLevel 解析日志级别
func ParseLogLevel(levelStr string) logrus.Level {
	level, err := logrus.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		return logrus.InfoLevel
	}
	return level
}

// WithField 添加字段到日志上下文
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		name:     l.name,
		logger:   l.logger,
		fileHook: l.fileHook,
		config:   l.config,
		Stats:    l.Stats,
		fields:   make(map[string]interface{}),
	}

	// 复制原有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	newLogger.fields[key] = value
	return newLogger
}

// WithFields 批量添加字段到日志上下文
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		name:     l.name,
		logger:   l.logger,
		fileHook: l.fileHook,
		config:   l.config,
		Stats:    l.Stats,
		fields:   make(map[string]interface{}),
	}

	// 复制原有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// getEntry 获取带有上下文的日志条目
func (l *Logger) getEntry() *logrus.Entry {
	entry := l.logger.WithField("app", l.name)

	l.mu.RLock()
	defer l.mu.RUnlock()

	for key, value := range l.fields {
		entry = entry.WithField(key, value)
	}

	return entry
}

// 基础日志方法
func (l *Logger) Log(level logrus.Level, args ...interface{}) {
	l.getEntry().Log(level, args...)
}

func (l *Logger) Logf(level logrus.Level, format string, args ...interface{}) {
	l.getEntry().Logf(level, format, args...)
}

func (l *Logger) Logln(level logrus.Level, args ...interface{}) {
	l.getEntry().Logln(level, args...)
}

// 各级别日志方法
func (l *Logger) Debug(args ...interface{}) {
	l.Log(logrus.DebugLevel, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logf(logrus.DebugLevel, format, args...)
}

func (l *Logger) Debugln(args ...interface{}) {
	l.Logln(logrus.DebugLevel, args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.Log(logrus.InfoLevel, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logf(logrus.InfoLevel, format, args...)
}

func (l *Logger) Infoln(args ...interface{}) {
	l.Logln(logrus.InfoLevel, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.Log(logrus.WarnLevel, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logf(logrus.WarnLevel, format, args...)
}

func (l *Logger) Warnln(args ...interface{}) {
	l.Logln(logrus.WarnLevel, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.Log(logrus.ErrorLevel, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(logrus.ErrorLevel, format, args...)
}

func (l *Logger) Errorln(args ...interface{}) {
	l.Logln(logrus.ErrorLevel, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.Log(logrus.FatalLevel, args...)
	l.logger.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logf(logrus.FatalLevel, format, args...)
	l.logger.Exit(1)
}

func (l *Logger) Fatalln(args ...interface{}) {
	l.Logln(logrus.FatalLevel, args...)
	l.logger.Exit(1)
}

func (l *Logger) Panic(args ...interface{}) {
	l.Log(logrus.PanicLevel, args...)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Logf(logrus.PanicLevel, format, args...)
}

func (l *Logger) Panicln(args ...interface{}) {
	l.Logln(logrus.PanicLevel, args...)
}

// 带错误字段的日志方法
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.getEntry().WithError(err)
}

// 打印统计信息
func (l *Logger) PrintStats() {
	if l.Stats == nil {
		l.Warn("统计功能未启用")
		return
	}

	fmt.Println("\n=== 运行统计 ===")
	if err := l.Stats.OutTableInfo(os.Stdout); err != nil {
		l.Errorf("输出统计信息失败: %v", err)
	}
}

// 设置日志级别
func (l *Logger) SetLevel(level string) {
	l.logger.SetLevel(ParseLogLevel(level))
}

// 获取日志级别
func (l *Logger) GetLevel() logrus.Level {
	return l.logger.GetLevel()
}

// 关闭日志文件
func (l *Logger) Close() error {
	if l.fileHook != nil {
		return l.fileHook.Close()
	}
	return nil
}

// 旋转日志文件（手动触发）
func (l *Logger) Rotate() error {
	if l.fileHook != nil {
		return l.fileHook.Rotate()
	}
	return nil
}

// DefaultLogger 默认日志记录器
var (
	defaultLogger *Logger
	once          sync.Once
)

// InitDefaultLogger 初始化默认日志记录器
func InitDefaultLogger(config *LogConfig) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(config)
	})
	return err
}

// GetDefaultLogger 获取默认日志记录器
func GetDefaultLogger() *Logger {
	if defaultLogger == nil {
		// 使用默认配置
		config := &LogConfig{
			AppName:       "default",
			LogLevel:      "info",
			EnableConsole: true,
			EnableFile:    false,
			EnableStats:   true,
		}
		defaultLogger, _ = NewLogger(config)
	}
	return defaultLogger
}

// 全局便捷方法
func Debug(args ...interface{}) {
	GetDefaultLogger().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetDefaultLogger().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetDefaultLogger().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetDefaultLogger().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	GetDefaultLogger().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

func WithField(key string, value interface{}) *Logger {
	return GetDefaultLogger().WithField(key, value)
}

func WithFields(fields map[string]interface{}) *Logger {
	return GetDefaultLogger().WithFields(fields)
}
