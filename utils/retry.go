package utils

import (
	"go.uber.org/zap"
	"time"
)

func Retry(fn func() error, maxRetry int) error {
	var err error
	var sleepTime = time.Second
	for i := 0; i < maxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		zap.L().Warn("try failed", zap.Int("retry", i), zap.Error(err))
		time.Sleep(sleepTime)
		sleepTime *= 2
	}
	return err
}
