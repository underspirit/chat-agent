//go:build ignore

package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const logTmFmtWithMS = "2006-01-02 15:04:05.000"

type Options struct {
	Level        string
	Format       string
	Director     string
	LinkName     string
	ShowLine     bool
	EncodeLevel  string
	LogInConsole bool
	Compress     bool
	MaxAge       int // 日志保留时长，天
	MaxSize      int // 日志保留空间，MB
	RotationTime int // 日志文件周期，天
	MaxBackup    int // 备份个数
}

type XLogger struct {
	*zap.Logger
	level        zapcore.Level
	format       string
	director     string
	linkName     string
	showLine     bool
	encodeLevel  string
	logInConsole bool
	compress     bool
	maxSize      int
	maxAge       int
	rotationTime int
	maxBackup    int
}

func Default() (*XLogger, error) {
	return New(Options{})
}

func New(options Options) (*XLogger, error) {
	xl := initOptions(options)

	if ok, _ := pathExists(xl.director); !ok {
		_ = os.Mkdir(xl.director, os.ModePerm)
	}

	core, err := xl.getEncoderCore()
	if err != nil {
		return nil, err
	}

	var logger *zap.Logger
	if xl.level == zap.DebugLevel || xl.level == zap.ErrorLevel {
		logger = zap.New(core, zap.AddStacktrace(xl.level))
	} else {
		logger = zap.New(core)
	}
	if xl.showLine {
		logger = logger.WithOptions(zap.AddCaller())
	}

	xl.Logger = logger

	return xl, nil
}

func initOptions(options Options) *XLogger {
	xl := &XLogger{}

	var level zapcore.Level
	switch options.Level {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "dpanic":
		level = zap.DPanicLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}
	xl.level = level

	if len(options.Format) == 0 {
		xl.format = "json" // default value
	} else {
		xl.format = options.Format
	}

	if len(options.Director) == 0 {
		xl.director = "log"
	} else {
		xl.director = options.Director
	}

	xl.linkName = options.LinkName
	xl.showLine = options.ShowLine
	xl.logInConsole = options.LogInConsole

	if len(options.EncodeLevel) == 0 {
		xl.encodeLevel = "LowercaseLevelEncoder"
	} else {
		xl.encodeLevel = options.EncodeLevel
	}

	if options.MaxAge == 0 {
		xl.maxAge = 7 * 24 // 默认保留7天
	} else {
		xl.maxAge = options.MaxAge
	}

	if options.MaxSize != 0 {
		xl.maxSize = options.MaxSize
	} else {
		xl.maxSize = 100 // 默认100M
	}
	xl.compress = options.Compress
	xl.maxBackup = options.MaxBackup

	if options.RotationTime == 0 {
		xl.rotationTime = 24 // 默认一天一个文件
	} else {
		xl.rotationTime = options.RotationTime
	}

	return xl
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (xl *XLogger) getEncoderConfig() (config zapcore.EncoderConfig) {
	config = zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 精简日志，不使用完整路径
	}
	switch xl.encodeLevel {
	case "LowercaseLevelEncoder": // 小写编码器(默认)
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	case "LowercaseColorLevelEncoder": // 小写编码器带颜色
		config.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	case "CapitalLevelEncoder": // 大写编码器
		config.EncodeLevel = zapcore.CapitalLevelEncoder
	case "CapitalColorLevelEncoder": // 大写编码器带颜色
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	return config
}

func (xl *XLogger) getEncoder() zapcore.Encoder {
	if xl.format == "json" {
		return zapcore.NewJSONEncoder(xl.getEncoderConfig())
	}
	return zapcore.NewConsoleEncoder(xl.getEncoderConfig())
}

func (xl *XLogger) getEncoderCore() (core zapcore.Core, err error) {
	// 使用file-rotatelogs 进行日志分割
	writer, err := xl.getWriteSyncer()
	if err != nil {
		return
	}
	return zapcore.NewCore(xl.getEncoder(), writer, xl.level), nil
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func (xl *XLogger) getWriteSyncer() (zapcore.WriteSyncer, error) {
	logWriter := &lumberjack.Logger{
		Filename:   fmt.Sprintf("./%s/sql.log", xl.director),
		MaxSize:    xl.maxSize,
		MaxAge:     xl.maxAge,
		MaxBackups: xl.maxBackup,
		Compress:   xl.compress,
		LocalTime:  true,
	}

	if xl.logInConsole {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(logWriter)), nil
	}
	return zapcore.AddSync(logWriter), nil
}
