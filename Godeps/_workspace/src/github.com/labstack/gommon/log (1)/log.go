package log

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/gommon/color"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

type (
	Logger struct {
		level  Level
		out    io.Writer
		err    io.Writer
		prefix string
		mu     sync.Mutex
	}
	Level uint8
)

const (
	TRACE = iota
	DEBUG
	INFO
	NOTICE
	WARN
	ERROR
	FATAL
	OFF
)

var (
	global = New("-")
	levels []string
)

func New(prefix string) (l *Logger) {
	l = &Logger{
		level:  INFO,
		prefix: prefix,
		out:    colorable.NewColorableStdout(),
		err:    colorable.NewColorableStderr(),
	}
	return
}

func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

func (l *Logger) SetLevel(v Level) {
	l.level = v
}

func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
	l.err = w

	switch w := w.(type) {
	case *os.File:
		if isatty.IsTerminal(w.Fd()) {
			color.Enable()
		}
	default:
		color.Disable()
	}

	// NOTE: Reintialize levels to reflect color enable/disable call.
	initLevels()
}

func (l *Logger) Print(msg interface{}, args ...interface{}) {
	f := fmt.Sprintf("%s", msg)
	fmt.Fprintf(l.out, f, args...)
}

func (l *Logger) Println(msg interface{}, args ...interface{}) {
	f := fmt.Sprintf("%s\n", msg)
	fmt.Fprintf(l.out, f, args...)
}

func (l *Logger) Trace(msg interface{}, args ...interface{}) {
	l.log(TRACE, l.out, msg, args...)
}

func (l *Logger) Debug(msg interface{}, args ...interface{}) {
	l.log(DEBUG, l.out, msg, args...)
}

func (l *Logger) Info(msg interface{}, args ...interface{}) {
	l.log(INFO, l.out, msg, args...)
}

func (l *Logger) Notice(msg interface{}, args ...interface{}) {
	l.log(NOTICE, l.out, msg, args...)
}

func (l *Logger) Warn(msg interface{}, args ...interface{}) {
	l.log(WARN, l.out, msg, args...)
}

func (l *Logger) Error(msg interface{}, args ...interface{}) {
	l.log(ERROR, l.err, msg, args...)
}

func (l *Logger) Fatal(msg interface{}, args ...interface{}) {
	l.log(FATAL, l.err, msg, args...)
}

func SetPrefix(p string) {
	global.SetPrefix(p)
}

func SetLevel(v Level) {
	global.SetLevel(v)
}

func SetOutput(w io.Writer) {
	global.SetOutput(w)
}

func Print(msg interface{}, args ...interface{}) {
	global.Print(msg, args...)
}

func Println(msg interface{}, args ...interface{}) {
	global.Println(msg, args...)
}

func Trace(msg interface{}, args ...interface{}) {
	global.Trace(msg, args...)
}

func Debug(msg interface{}, args ...interface{}) {
	global.Debug(msg, args...)
}

func Info(msg interface{}, args ...interface{}) {
	global.Info(msg, args...)
}

func Notice(msg interface{}, args ...interface{}) {
	global.Notice(msg, args...)
}

func Warn(msg interface{}, args ...interface{}) {
	global.Warn(msg, args...)
}

func Error(msg interface{}, args ...interface{}) {
	global.Error(msg, args...)
}

func Fatal(msg interface{}, args ...interface{}) {
	global.Fatal(msg, args...)
}

func (l *Logger) log(v Level, w io.Writer, msg interface{}, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if v >= l.level {
		// TODO: Improve performance
		f := fmt.Sprintf("%s|%s|%v\n", levels[v], l.prefix, msg)
		fmt.Fprintf(w, f, args...)
	}
}

func initLevels() {
	levels = []string{
		color.Cyan("TRACE"),
		color.Blue("DEBUG"),
		color.Green("INFO"),
		color.Magenta("NOTICE"),
		color.Yellow("WARN"),
		color.Red("ERROR"),
		color.RedBg("FATAL"),
	}
}

func init() {
	initLevels()
}
