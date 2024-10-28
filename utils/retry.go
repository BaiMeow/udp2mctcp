package utils

import (
	"go.uber.org/zap"
)

func Retry(fn func() error, maxRetry int) error {
	var err error
	for i := 0; i < maxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		zap.L().Warn("try failed", zap.Int("retry", i), zap.Error(err))
	}
	return err
}
