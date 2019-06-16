package log

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	stdLogger  = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	fileLogger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
)

func init() {
	initStdLogger()
}

// default std logger is enabled
func EnableStdLogger(enable bool) {
	if enable && stdLogger == nil {
		initStdLogger()
	}
	if !enable {
		stdLogger = nil
	}
}

// default file logger is disabled
func EnableFileLogger(enable bool, savePath string) error {
	if enable {
		fileLogger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		err := initFileLogger(savePath)
		if err != nil {
			fileLogger = nil
		}
		return err
	} else {
		fileLogger = nil
		return nil
	}
}

func EnableOnlyFileLogger(enable bool, savePath string) error {
	if enable {
		fileLogger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		err := initFileLogger(savePath)
		if err != nil {
			fileLogger = nil
		}
		stdLogger = nil
		return err
	} else {
		fileLogger = nil
		return nil
	}
}

func Debug(keyvals ...interface{}) {
	if stdLogger != nil {
		level.Debug(stdLogger).Log(keyvals)
	}
	if fileLogger != nil {
		level.Debug(fileLogger).Log(keyvals)
	}
}

func Info(keyvals ...interface{}) {
	if stdLogger != nil {
		level.Info(stdLogger).Log(keyvals)
	}
	if fileLogger != nil {
		level.Info(fileLogger).Log(keyvals)
	}
}

func Warn(keyvals ...interface{}) {
	if stdLogger != nil {
		level.Warn(stdLogger).Log(keyvals)
	}
	if fileLogger != nil {
		level.Warn(fileLogger).Log(keyvals)
	}
}

func Error(keyvals ...interface{}) {
	if stdLogger != nil {
		level.Error(stdLogger).Log(keyvals)
	}
	if fileLogger != nil {
		level.Error(fileLogger).Log(keyvals)
	}
}

func SetToDebug() {
	if stdLogger != nil {
		stdLogger = level.NewFilter(stdLogger, level.AllowDebug())
	}
	if fileLogger != nil {
		fileLogger = level.NewFilter(fileLogger, level.AllowDebug())
	}
}

func SetToInfo() {
	if stdLogger != nil {
		stdLogger = level.NewFilter(stdLogger, level.AllowInfo())
	}
	if fileLogger != nil {
		fileLogger = level.NewFilter(fileLogger, level.AllowInfo())
	}
}

func SetToWarn() {
	if stdLogger != nil {
		stdLogger = level.NewFilter(stdLogger, level.AllowWarn())
	}
	if fileLogger != nil {
		fileLogger = level.NewFilter(fileLogger, level.AllowWarn())
	}
}

func SetToError() {
	if stdLogger != nil {
		stdLogger = level.NewFilter(stdLogger, level.AllowError())
	}
	if fileLogger != nil {
		fileLogger = level.NewFilter(fileLogger, level.AllowError())
	}
}

func initStdLogger() {
	stdLogger = log.With(stdLogger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
}

func initFileLogger(savePath string) error {
	if _, err := os.Stat(savePath); err != nil {
		err = os.MkdirAll(filepath.Dir(savePath), 0777)
		if err != nil {
			return err
		}
		_, err = os.Create(savePath)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY, 0777)
	if err == nil {
		fileLogger = log.NewLogfmtLogger(io.MultiWriter(file))
		fileLogger = log.With(fileLogger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	} else {
		return err
	}

	return nil
}
