package electrum

import (
	"fmt"
	"log"
)

type logger struct {
	*log.Logger
}

func (l *logger) Debugf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
}

func (l *logger) Infof(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf("[INFO] "+format, v...))
}

func (l *logger) Warnf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf("[WARN] "+format, v...))
}

func (l *logger) Errorf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
}

func newLogger() Logger {
	return &logger{
		log.New(log.Writer(), "", log.LstdFlags|log.Lshortfile),
	}
}
