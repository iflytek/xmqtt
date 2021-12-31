package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
	"sync"
	"time"
)

// logger
var (
	logger *zap.SugaredLogger
	once   sync.Once
	dynamicLevel = zap.NewAtomicLevel()
	Log *zap.SugaredLogger
)

// 格式化时间
func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func NewEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        "Time",                        // json时时间键
		LevelKey:       "Level",                       // json时日志等级键
		NameKey:        "Name",                        // json时日志记录器名
		CallerKey:      "Caller",                      // json时日志文件信息键
		MessageKey:     "Message",                     // json时日志消息键
		StacktraceKey:  "StackTrace",                  // json时堆栈键
		LineEnding:     zapcore.DefaultLineEnding,     // 友好日志换行符
		EncodeLevel:    zapcore.CapitalLevelEncoder,   // 友好日志等级名大小写（info INFO）
		EncodeTime:     TimeEncoder,                   // 友好日志时日期格式化
		EncodeDuration: zapcore.StringDurationEncoder, // 时间序列化
		EncodeCaller:   zapcore.ShortCallerEncoder,    // 日志文件信息（包/文件.go:行号）
		EncodeName:     zapcore.FullNameEncoder,
	}
}

func NewLoggerRotate(path string) *zap.SugaredLogger{
	return NewLogger(path, "info")
}

func SetLevel(level zapcore.Level) {
	dynamicLevel.SetLevel(level)
}

/**
@param path 日志存储路径
@param level 日志等级 debug error warn info
*/
func NewLogger(path string, level string) *zap.SugaredLogger {

	if path == "" {
		path = "aiam.log"
	}

	once.Do(func() {
		hook := lumberjack.Logger{
			Filename:   path,
			MaxSize:    100,  // 日志文件最大大小（单位：MB）,10M
			MaxBackups: 10, // 保留的旧日志文件最大数量
			MaxAge:     30, // 保存日期
			LocalTime:  true,
			Compress:   false,
		}
		defer hook.Close()

		w := zapcore.AddSync(&hook)

		// 设置日志级别
		var logLevel zapcore.Level
		switch level {
		case "debug":
			logLevel = zap.DebugLevel
		case "warn":
			logLevel = zap.WarnLevel
		case "error":
			logLevel = zap.ErrorLevel
		case "info":
			logLevel = zap.InfoLevel
		}

		dynamicLevel.SetLevel(logLevel)
		encoderConfig := NewEncoderConfig()
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			w,
			dynamicLevel,
		)

		log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		logger = log.Sugar()
		Log = logger
	})

	return logger
}

func NewLoggerInConsole(level string) *zap.SugaredLogger {
	once.Do(func() {
		w := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))

		var logLevel zapcore.Level
		switch level {
		case "debug":
			logLevel = zap.DebugLevel
		case "warn":
			logLevel = zap.WarnLevel
		case "error":
			logLevel = zap.ErrorLevel
		case "info":
			logLevel = zap.InfoLevel
		}

		dynamicLevel.SetLevel(logLevel)
		encoderConfig := NewEncoderConfig()
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			w,
			dynamicLevel,
		)

		log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		logger = log.Sugar()
		Log = logger
	})
	return logger
}

func GetLevel(level string) (zapcore.Level, error) {
	var logLevel zapcore.Level
	level = strings.ToUpper(level)
	if level == "DEBUG" {
		logLevel = zapcore.DebugLevel
		return logLevel, nil
	} else if level == "INFO" {
		logLevel = zapcore.InfoLevel
		return logLevel, nil
	} else if level == "WARN" {
		logLevel = zapcore.WarnLevel
		return logLevel, nil
	} else if level == "ERROR" {
		logLevel = zapcore.ErrorLevel
		return logLevel, nil
	} else if level == "DPANIC" {
		logLevel = zapcore.DPanicLevel
		return logLevel, nil
	} else if level == "PANIC" {
		logLevel = zapcore.PanicLevel
		return logLevel, nil
	} else if level == "FATAL" {
		logLevel = zapcore.FatalLevel
		return logLevel, nil
	} else {
		return -1, fmt.Errorf("invalid log level")
	}
}


func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

func Debugln(args ...interface{}) {
	logger.Debug(args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

func Infoln(args ...interface{}) {
	logger.Info(args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	logger.Infow(msg, keysAndValues...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

func Warnln(args ...interface{}) {
	logger.Warn(args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	logger.Warnw(msg, keysAndValues...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

func Errorln(args ...interface{}) {
	logger.Error(args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	logger.Errorw(msg, keysAndValues...)
}

func DPanic(args ...interface{}) {
	logger.DPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
	logger.DPanicf(template, args...)
}

func DPanicw(msg string, keysAndValues ...interface{}) {
	logger.DPanicw(msg, keysAndValues...)
}

func Panic(args ...interface{}) {
	logger.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	logger.Panicf(template, args...)
}

func Panicln(args ...interface{}) {
	logger.Panic(args...)
}

func Panicw(msg string, keysAndValues ...interface{}) {
	logger.Panicw(msg, keysAndValues...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}

func Fatalln(args ...interface{}) {
	logger.Fatal(args...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	logger.Fatalw(msg, keysAndValues...)
}
