package utils

import (
	"os"

	"github.com/seniorGolang/gokit/logger"
)

func ExitOnError(log logger.Logger, err error, msg string) {
	if err != nil {
		log.WithError(err).Error(msg)
		os.Exit(1)
	}
}
