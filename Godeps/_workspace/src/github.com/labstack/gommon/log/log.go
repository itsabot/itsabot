package log

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/gommon/color"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/mattn/go-colorable"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/mattn/go-isatty"
)

type (
	Logger struct {
		level  Level
		out    io.Writer
		prefix string
		mu     sync.Mutex
	}
	Level uint8
)

const (
	DEBUG = iota
	INFO
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
	}
	l.SetOutput(colorable.NewColorableStdout())
	return
}

func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

func (l *Logger) SetLevel(v Level) {
	l.level = v
}

func (l *Logger) Level() Level {
	return l.level
}

func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
	color.Disable()

	if w, ok := w.(*os.File); ok && isatty.IsTerminal(w.Fd()) {
		color.Enable()
	}

	// NOTE: Reintialize levels to reflect color enable/disable call.
	initLevels()
}

func (l *Logger) Print(i ...interface{}) {
	fmt.Println(i...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.out, f, args...)
}

func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
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

func Print(i ...interface{}) {
	global.Print(i...)
}

func Printf(format string, args ...interface{}) {
	global.Printf(format, args...)
}

func Debug(i ...interface{}) {
	global.Debug(i...)
}

func Debugf(format string, args ...interface{}) {
	global.Debugf(format, args...)
}

func Info(i ...interface{}) {
	global.Info(i...)
}

func Infof(format string, args ...interface{}) {
	global.Infof(format, args...)
}

func Warn(i ...interface{}) {
	global.Warn(i...)
}

func Warnf(format string, args ...interface{}) {
	global.Warnf(format, args...)
}

func Error(i ...interface{}) {
	global.Error(i...)
}

func Errorf(format string, args ...interface{}) {
	global.Errorf(format, args...)
}

func Fatal(i ...interface{}) {
	global.Fatal(i...)
}

func Fatalf(format string, args ...interface{}) {
	global.Fatalf(format, args...)
}

func (l *Logger) log(v Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if v >= l.level {
		if format == "" {
			fmt.Println(args...)
		} else {
			// TODO: Improve performance
			f := fmt.Sprintf("%s|%s|%v\n", levels[v], l.prefix, format)
			fmt.Fprintf(l.out, f, args...)
		}
	}
}

func initLevels() {
	levels = []string{
		color.Blue("DEBUG"),
		color.Green("INFO"),
		color.Yellow("WARN"),
		color.Red("ERROR"),
		color.RedBg("FATAL"),
	}
}

func init() {
	initLevels()
}
