package utils

import "log"

func Retry(fn func() error, maxRetry int) error {
	var err error
	for i := 0; i < maxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Printf("retry %d, err: %v", i, err)
	}
	return err
}
