package logger

import (
	"fmt"
	"os"
)

var dbgprefix = "DEBUG:>> "
var dbgon = false

func DbgPrefix(s string) {
	dbgprefix = s
}

func DbgOn(b bool) {
	dbgon = b
}

func Dbg(v ...interface{}) {
	Dbgf("%s", fmt.Sprintln(v...))
}

func Dbgf(format string, v ...interface{}) {
	if dbgon {
		Putf(dbgprefix+format, v...)
	}
}

func Ftl(code int, v ...interface{}) {
	Ftlf(code, "%s", fmt.Sprintln(v...))
}

func Ftlf(code int, format string, v ...interface{}) {
	Putf(format, v...)
	os.Exit(code)
}

func Put(v ...interface{}) {
	Putf("%s", fmt.Sprintln(v...))
}

func Putf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if _, err := fmt.Print(msg); err == nil {
		return
	}
	if _, err := fmt.Fprint(os.Stderr, "[stdout failed] "+msg); err == nil {
		return
	}
	md := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	f, err := os.OpenFile("./last_resort.log", md, 0600)
	if err != nil {
		return
	}
	f.WriteString("[stderr failed] " + msg)
	f.Close()
}
