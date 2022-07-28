package conf

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

var (
	logInitMsg string
	level      zapcore.Level
)

func LoadGlobalLogger() {
	zapConf := &zap.Config{
		Level:             zap.AtomicLevel{},
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "",
		EncoderConfig:     zapcore.EncoderConfig{},
		OutputPaths:       nil,
		ErrorOutputPaths:  nil,
		InitialFields:     nil,
	}
}

func loadGlobalLogger() error {
	Logger := zap.Config{}
	// 加载日志等级
	logLevel, err := zap.NewLevel(string(configLog.Level))
	if err != nil {
		logInitMsg = fmt.Sprintf("%s, use default level INFO", err)
		level = zap.InfoLevel
	} else {
		level = logLevel
		logInitMsg = fmt.Sprintf("log level: %s", logLevel)
	}

	// 使用默认配置初始化全局Logger
	zapConfig := zap.DefaultConfig()
	// 配置log level
	zapConfig.Level = level
	// 程序每启动一次，不必每次都生成一个日志文件
	zapConfig.Files.RotateOnStartup = false
	// 配置文件输出方式
	switch configLog.To {
	case conf.ToStdout:
		// 把日志打印到标准输出
		zapConfig.ToStderr = true
		// 并没在把日志输入输出到文件
		zapConfig.ToFiles = false
	case conf.ToFile:
		zapConfig.Files.Name = "api.log"
		zapConfig.Files.Path = configLog.PathDir
	}

	// 配置日志的输出格式
	switch configLog.Format {
	case conf.JSONFormat:
		zapConfig.JSON = true
	}

	// 把配置用于全局Logger
	if err := zap.Configure(zapConfig); err != nil {
		return err
	}
	zap.L().Named("INIT").Info(logInitMsg)
	return nil
}
