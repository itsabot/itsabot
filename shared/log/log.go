package log

import (
	"fmt"
	"log"
)

var debugprefix = "DEBUG: "
var debugon = false

func DebugPrefix(s string) {
	debugprefix = s
}

func DebugOn(b bool) {
	debugon = b
}

func Debug(v ...interface{}) {
	Debugf("%s", fmt.Sprintln(v...))
}

func Debugf(format string, v ...interface{}) {
	if debugon {
		Infof(debugprefix+format, v...)
	}
}

func Info(v ...interface{}) {
	Infof("%s", fmt.Sprintln(v...))
}

func Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print(msg)
}

// Warn is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func Warn(v ...interface{}) {
	Debug(v...)
}

// Warnf is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func Warnf(format string, v ...interface{}) {
	Debugf(format, v...)
}

// Error is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func Error(v ...interface{}) {
	Debug(v...)
}

// Errorf is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func Errorf(format string, v ...interface{}) {
	Debugf(format, v...)
}

func Fatal(v ...interface{}) {
	Fatalf("%s", fmt.Sprintln(v...))
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Logger is defined for packages to use in place of the stdlib log. It sets a
// prefix of the package name with each log and follows the convention of.
type Logger struct {
	prefix string
}

func New(pkgName string) *Logger {
	return &Logger{
		prefix: fmt.Sprintf("[%s] ", pkgName),
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.Debugf("%s", fmt.Sprintln(v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if debugon {
		l.Infof(debugprefix+l.prefix+format, v...)
	}
}

func (l *Logger) Info(v ...interface{}) {
	l.Infof("%s", fmt.Sprintln(v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print(msg)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.Fatalf("%s", l.prefix+fmt.Sprintln(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}
