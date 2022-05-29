package logger

import (
	"sync"

	"go.uber.org/zap"
)

var logger *zap.Logger
var once sync.Once

// GetZapLogger returns an instance of zap logger
func GetZapLogger() (*zap.Logger, error) {
	var err error
	once.Do(func() {
		logger, err = zap.NewDevelopment()
		// logger, err = zap.NewProduction()
	})

	return logger, err
}
