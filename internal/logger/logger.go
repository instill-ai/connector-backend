package logger

import (
	"sync"

	"github.com/instill-ai/connector-backend/config"
	"go.uber.org/zap"
)

var logger *zap.Logger
var once sync.Once

// GetZapLogger returns an instance of zap logger
func GetZapLogger() (*zap.Logger, error) {
	var err error
	once.Do(func() {
		if config.Config.Server.Debug {
			logger, err = zap.NewDevelopment()
		} else {
			logger, err = zap.NewProduction()
		}
	})

	return logger, err
}
