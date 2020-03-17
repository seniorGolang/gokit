package logger

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/gokit/logger/format"
)

var Log Logger
type Logger = *logrus.Entry
type Fields = logrus.Fields
type FieldMap = logrus.FieldMap

func init() {
	Log = logrus.WithTime(time.Now())
	logrus.SetFormatter(&format.Formatter{TimestampFormat: time.StampMilli})
}
