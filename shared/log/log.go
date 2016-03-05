package log

import (
	"fmt"
	"log"
	"os"
)

var debugprefix = "DEBUG: "
var debugon = false

func DebugPrefix(s string) {
	debugprefix = s
}

func SetDebug(b bool) {
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
func (l *Logger) Warn(v ...interface{}) {
	Debug(v...)
}

// Warnf is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func (l *Logger) Warnf(format string, v ...interface{}) {
	Debugf(format, v...)
}

// Error is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func (l *Logger) Error(v ...interface{}) {
	Debug(v...)
}

// Errorf is not used, but it's included to satisfy the Echo router's Logger
// interface. The rationale on why Warn and Error have been excluded can be
// found here: http://dave.cheney.net/2015/11/05/lets-talk-about-logging
func (l *Logger) Errorf(format string, v ...interface{}) {
	Debugf(format, v...)
}

func Fatal(v ...interface{}) {
	Fatalf("%s", fmt.Sprintln(v...))
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Logger is defined for plugins to use in place of the stdlib log. It sets a
// prefix of the plugin name with each log and follows the convention of.
type Logger struct {
	prefix string
	logger *log.Logger
	debug  bool
}

func New(pkgName string) *Logger {
	lg := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	if len(pkgName) == 0 {
		return &Logger{logger: lg}
	}
	return &Logger{
		prefix: fmt.Sprintf("[%s] ", pkgName),
		logger: lg,
	}
}

func (l *Logger) SetDebug(b bool) {
	l.debug = b
}

func (l *Logger) Debug(v ...interface{}) {
	l.Debugf("%s", fmt.Sprintln(v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.debug {
		l.Infof(debugprefix+l.prefix+format, v...)
	}
}

func (l *Logger) Info(v ...interface{}) {
	l.Infof("%s", fmt.Sprintln(v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Printf(msg)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Fatalf("%s", l.prefix+fmt.Sprintln(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
}

func (l *Logger) SetFlags(flag int) {
	l.logger.SetFlags(flag)
}
