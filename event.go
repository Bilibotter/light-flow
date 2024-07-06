package light_flow

import (
	"log"
	"os"
)

var (
	logger LoggerI = newDefaultLogger()
)

type methodNotSupport struct{}

// LoggerI 定义接口
type LoggerI interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

type queue[T any] interface {
	Enqueue(T) bool
	Dequeue() (T, bool)
	Len() int
}

type eventHandler struct {
}

// defaultLogger 结构体，嵌入 log.Logger
type defaultLogger struct {
	*log.Logger
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		Logger: log.New(os.Stdout, "[light-flow] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(v ...interface{}) {
	panic(methodNotSupport{})
}

func (l *defaultLogger) Info(v ...interface{}) {
	panic(methodNotSupport{})
}

func (l *defaultLogger) Warn(v ...interface{}) {
	panic(methodNotSupport{})
}

func (l *defaultLogger) Error(v ...interface{}) {
	panic(methodNotSupport{})
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.Printf("[DEBUG] "+format+"\n", v...)
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.Printf("[INFO] "+format+"\n", v...)
}

func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.Printf("[WARN] "+format+"\n", v...)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.Printf("[ERROR] "+format+"\n", v...)
}
