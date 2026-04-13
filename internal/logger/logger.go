package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var L = logrus.New()

func init() {
	L.SetReportCaller(false)
	L.SetLevel(logrus.InfoLevel)

	L.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05.000",
	})

	L.SetOutput(os.Stdout)
}

func formatCaller(file string, line int) string {
	path := filepath.ToSlash(file)
	for _, prefix := range []string{"/internal/", "/cmd/"} {
		if idx := strings.Index(path, prefix); idx >= 0 {
			return path[idx+1:] + ":" + fmt.Sprintf("%d", line)
		}
	}
	_, name := filepath.Split(path)
	return name + ":" + fmt.Sprintf("%d", line)
}

func withCaller() *logrus.Entry {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return L.WithField("file", "unknown")
	}
	return L.WithField("file", formatCaller(file, line))
}

func SetOutput(w io.Writer) {
	L.SetOutput(w)
}

func Writer() io.Writer {
	return L.Out
}

func Infof(format string, args ...interface{}) {
	withCaller().Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	withCaller().Errorf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	withCaller().Warnf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	withCaller().Debugf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	withCaller().Fatalf(format, args...)
}

func Fatal(args ...interface{}) {
	withCaller().Fatal(args...)
}

func Info(args ...interface{}) {
	withCaller().Info(args...)
}

func Debug(args ...interface{}) {
	withCaller().Debug(args...)
}

func Warn(args ...interface{}) {
	withCaller().Warn(args...)
}

func Error(args ...interface{}) {
	withCaller().Error(args...)
}
