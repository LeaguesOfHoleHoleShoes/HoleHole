
// default logger
package log

import (
	"go.uber.org/zap"
)

func init() {
	// call InitLog outside if need change cfg
	InitLog(DefaultDebugCfg())
}

var L *zap.Logger

func InitLog(cfg zap.Config) {
	var err error
	if L, err = cfg.Build(); err != nil {
		panic(err)
	}
}

func DefaultDebugCfg() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	// set log output
	cfg.OutputPaths = []string{"stdout"}
	//cfg.ErrorOutputPaths = []string{logPath}
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	return cfg
}

func DefaultProdCfg() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}
	//cfg.ErrorOutputPaths = []string{}
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	return cfg
}
