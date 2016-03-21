// Package log is a logger used by Abot core and plugins. It standardizes
// logging formats consistent with Abot's needs, allowing for a debug mode, and
// enabling plugins to specify their own name so each plugin's logs are easy to
// isolate.
package log

import (
	"fmt"
	"log"
	"os"
)

var debugprefix = "DEBUG: "
var debugon = false

// DebugPrefix overrides the default "DEBUG: " prefix for debug logs.
func DebugPrefix(s string) {
	debugprefix = s
}

// SetDebug shows or hides debug logs. Default: false (debug off)
func SetDebug(b bool) {
	debugon = b
}

// Debug logs a statement with the debug prefix if SetDebug(true) has been
// called and ends with a new line.
func Debug(v ...interface{}) {
	Debugf("%s", fmt.Sprintln(v...))
}

// Debugf logs a statement with the debug prefix if SetDebug(true) has been
// called, allowing for custom formatting.
func Debugf(format string, v ...interface{}) {
	if debugon {
		Infof(debugprefix+format, v...)
	}
}

// Info logs a statement and ends with a new line.
func Info(v ...interface{}) {
	Infof("%s", fmt.Sprintln(v...))
}

// Infof logs a statement, allowing for custom formatting.
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

// Fatal logs a statement and kills the running process.
func Fatal(v ...interface{}) {
	Fatalf("%s", fmt.Sprintln(v...))
}

// Fatalf logs a statement and kills the running process, allowing for custom
// formatting.
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

// New returns a Logger which supports a custom prefix representing a plugin's
// name.
func New(pluginName string) *Logger {
	lg := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	if len(pluginName) == 0 {
		return &Logger{logger: lg}
	}
	return &Logger{
		prefix: fmt.Sprintf("[%s] ", pluginName),
		logger: lg,
	}
}

// SetDebug shows or hides debug logs. Default: false (debug off)
func (l *Logger) SetDebug(b bool) {
	l.debug = b
}

// Debug logs a statement with the debug prefix if SetDebug(true) has been
// called and ends with a new line.
func (l *Logger) Debug(v ...interface{}) {
	l.Debugf("%s", fmt.Sprintln(v...))
}

// Debugf logs a statement with the debug prefix if SetDebug(true) has been
// called, allowing for custom formatting.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.debug {
		l.Infof(debugprefix+l.prefix+format, v...)
	}
}

// Info logs a statement and ends with a new line.
func (l *Logger) Info(v ...interface{}) {
	l.Infof("%s", fmt.Sprintln(v...))
}

// Infof logs a statement, allowing for custom formatting.
func (l *Logger) Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Printf(msg)
}

// Fatal logs a statement and kills the running process.
func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Fatalf("%s", l.prefix+fmt.Sprintln(v...))
}

// Fatalf logs a statement and kills the running process, allowing for custom
// formatting.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
}

// SetFlags enables customizing flags just like the standard library's
// log.SetFlags.
func (l *Logger) SetFlags(flag int) {
	l.logger.SetFlags(flag)
}
