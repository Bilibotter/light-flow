package light_flow

import (
	"log"
	"os"
)

const (
	resourceErrorFmt = "Process[Name: %s, ID: %s] %s Resource[ %s ] failed;\nerror=%s"
	resourcePanicFmt = "Process[Name: %s, ID: %s] %s Resource[ %s ] panic;\npanic=%v\n%s"
	recoverLog       = "panic occur while WorkFlow[ %s ] recovering;\nID=%s\nPanic=%s\n%s"
	saveLog          = "panic occur while WorkFlow[ %s ] saving checkpoints;\nID=%s\nPanic=%s\n%s"
	callbackPanicLog = "%s Callback panic;\nID=%s;\nBelong=%s;\nNecessity=%s, Scope=%s, Iteration=%d;\nPanic=%v\n%s"
	callbackErrorLog = "%s Callback error;\nID=%s;\nBelong=%s;\nNecessity=%s, Scope=%s, Iteration=%d;\nError=%s"
	snapshotErrorLog = "build snapshot failed: %s[Name=%s, ID=%s] %s error: %s"
)

var (
	logger LoggerI = newDefaultLogger()
)

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

// defaultLogger
type defaultLogger struct {
	*log.Logger
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		Logger: log.New(os.Stdout, "[light-flow] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(_ ...interface{}) {
	panic("method not support")
}

func (l *defaultLogger) Info(_ ...interface{}) {
	panic("method not support")
}

func (l *defaultLogger) Warn(_ ...interface{}) {
	panic("method not support")
}

func (l *defaultLogger) Error(_ ...interface{}) {
	panic("method not support")
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
