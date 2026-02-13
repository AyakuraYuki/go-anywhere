package log

import (
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.StandardLogger()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		DisableQuote:              true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           time.DateTime,
		PadLevelText:              true,
	})
}

func Scope(name string) *logrus.Entry { return logger.WithField("scope", name) }
func Main() *logrus.Entry             { return Scope("anywhere") }

func Info(msg string)                  { logger.Info(msg) }
func Infof(format string, args ...any) { logger.Infof(format, args...) }

func Warn(msg string)                  { logger.Warn(msg) }
func Warnf(format string, args ...any) { logger.Warnf(format, args...) }

func Error(msg string)                  { logger.Error(msg) }
func Errorf(format string, args ...any) { logger.Errorf(format, args...) }

func Debug(msg string)                  { logger.Debug(msg) }
func Debugf(format string, args ...any) { logger.Debugf(format, args...) }

func Trace(msg string)                  { logger.Trace(msg) }
func Tracef(format string, args ...any) { logger.Tracef(format, args...) }
