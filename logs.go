package light_flow

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	callbackOrder = []string{"Position", "Order", "Scope", "Necessity"}
	suspendOrder  = []string{"Position"}
)

const (
	resourceErrorFmt = "Process[Name: %s, ID: %s] %s Resource[ %s ] failed;\nerror=%s"
	resourcePanicFmt = "Process[Name: %s, ID: %s] %s Resource[ %s ] panic;\npanic=%v\n%s"
	recoverLog       = "panic occur while WorkFlow[ %s ] recovering;\nID=%s\nPanic=%s\n%s"
	persistPanicLog  = "persist failed: %s[Name:%s, ID:%s] persist panic while %s\npanic: %v\n%s"
	persistErrorLog  = "persist failed: %s[Name:%s, ID:%s] persist error while %s: %s"
	errorLog         = "[Stage: %s] [%s: %s] %s[ID: %s] - Failed | Error: %s"
	panicLog         = "[Stage: %s] [%s: %s] %s[ID: %s] - Failed | Panic: %v\n%s"
)

const (
	execStage     = "execute"
	callbackStage = "callback"
)

const (
	notSupport = "method not support"
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

func commonLog(order []string) func(event FlexEvent) {
	return func(event FlexEvent) {
		sb := strings.Builder{}
		if event.ExtraInfo() != nil {
			for _, key := range order {
				sb.WriteString(fmt.Sprintf("[%s: %s] ", key, event.Extra(key)))
			}
		}
		if event.Level() == ErrorLevel {
			logger.Errorf(errorLog, event.Stage(), event.Layer(), event.Name(), sb.String(), event.ID(), event.Error())
			return
		}
		if event.Level() == PanicLevel {
			logger.Errorf(panicLog, event.Stage(), event.Layer(), event.Name(), sb.String(), event.ID(), event.Panic(), event.StackTrace())
		}
	}
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		Logger: log.New(os.Stdout, "[light-flow] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(_ ...interface{}) {
	panic(notSupport)
}

func (l *defaultLogger) Info(_ ...interface{}) {
	panic(notSupport)
}

func (l *defaultLogger) Warn(_ ...interface{}) {
	panic(notSupport)
}

func (l *defaultLogger) Error(_ ...interface{}) {
	panic(notSupport)
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
