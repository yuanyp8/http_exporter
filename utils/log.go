package utils

import (
	"go.uber.org/zap"
	"log"
	"sync"
)

var (
	Logger *zap.Logger
	once   = sync.Once{}
)

func loadGlobalLogger() {
	logger, err := zap.NewProduction()
	defer logger.Sync()
	if err != nil {
		log.Fatalf("set zap log production error, %s", err)
	}
	logger.Named("INIT").Debug("load global logger", zap.Bool("successful", true))
	Logger = logger
}

func init() {
	// 单次加载
	once.Do(loadGlobalLogger)
}
