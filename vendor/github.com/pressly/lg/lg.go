package lg

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
)

var (
	DefaultLogger *logrus.Logger = logrus.New()
	AlertFn       func(level logrus.Level, msg string)
)

func WithField(key string, value interface{}) *logrus.Entry {
	return DefaultLogger.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return DefaultLogger.WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return DefaultLogger.WithError(err)
}

func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	DefaultLogger.Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}

func Alertf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	DefaultLogger.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	DefaultLogger.Panicf(format, args...)
}

func Debug(args ...interface{}) {
	DefaultLogger.Debug(args...)
}

func Info(args ...interface{}) {
	DefaultLogger.Info(args...)
}

func Print(args ...interface{}) {
	DefaultLogger.Print(args...)
}

func Warn(args ...interface{}) {
	DefaultLogger.Warn(args...)
}

func Error(args ...interface{}) {
	DefaultLogger.Error(args...)
}

func Alert(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	Error(args...)
}

func Fatal(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	DefaultLogger.Fatal(args...)
}

func Panic(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	DefaultLogger.Panic(args...)
}

func Debugln(args ...interface{}) {
	DefaultLogger.Debugln(args...)
}

func Infoln(args ...interface{}) {
	DefaultLogger.Infoln(args...)
}

func Println(args ...interface{}) {
	DefaultLogger.Println(args...)
}

func Warnln(args ...interface{}) {
	DefaultLogger.Warnln(args...)
}

func Errorln(args ...interface{}) {
	DefaultLogger.Errorln(args...)
}

func Alertln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	Errorln(args...)
}

func Fatalln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	DefaultLogger.Fatalln(args...)
}

func Panicln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	DefaultLogger.Panicln(args...)
}
