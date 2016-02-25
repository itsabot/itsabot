package log

import (
	"fmt"
	"log"
)

var debugprefix = "DEBUG:>> "
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

func Fatal(v ...interface{}) {
	Fatalf("%s", fmt.Sprintln(v...))
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func Info(v ...interface{}) {
	Infof("%s", fmt.Sprintln(v...))
}

func Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print(msg)
}
