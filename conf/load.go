package conf

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"sync"
)

type SafeConfig struct {
	sync.RWMutex
	C *Config
}

func (sc *SafeConfig) ReloadConfig(configFile string) (err error) {
	c := &Config{}

	// 配置文件加载完成后记录一次加载状态
	defer func() {
		if err != nil {
			configReloadSuccess.Set(0)
		} else {
			configReloadSuccess.Set(1)
			configReloadSeconds.SetToCurrentTime()
		}
	}()

	// load config from the given pathname file
	vip := viper.New()
	vip.SetConfigFile(configFile)

	if err := vip.ReadInConfig(); err != nil {
		l.Error("error loading config", zap.String("filePath", configFile))
		return
	}
	if err := vip.Unmarshal(c); err != nil {
		l.Error("error unmarshal config", zap.String("filePath", configFile))
		return
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()
	return
}
