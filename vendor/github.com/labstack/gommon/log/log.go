package log

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"github.com/labstack/gommon/color"
)

type (
	Logger struct {
		level  Level
		levels []string
		color  color.Color
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
)

func New(prefix string) (l *Logger) {
	l = &Logger{
		level:  INFO,
		prefix: prefix,
	}
	l.SetOutput(colorable.NewColorableStdout())
	return
}

func (l *Logger) initLevels() {
	l.levels = []string{
		l.color.Blue("DEBUG"),
		l.color.Green("INFO"),
		l.color.Yellow("WARN"),
		l.color.Red("ERROR"),
		l.color.RedBg("FATAL"),
	}
}

func (l *Logger) DisableColor() {
	l.color.Disable()
	l.initLevels()
}

func (l *Logger) EnableColor() {
	l.color.Enable()
	l.initLevels()
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
	l.DisableColor()

	if w, ok := w.(*os.File); ok && isatty.IsTerminal(w.Fd()) {
		l.EnableColor()
	}
}

func (l *Logger) Print(i ...interface{}) {
	fmt.Fprintln(l.out, i...)
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

func DisableColor() {
	global.DisableColor()
}

func EnableColor() {
	global.EnableColor()
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
		if format == "" && len(args) > 0 {
			args[0] = fmt.Sprintf("%s|%s|%s", l.levels[v], l.prefix, args[0])
			fmt.Fprintln(l.out, args...)
		} else {
			// TODO: Improve performance
			f := fmt.Sprintf("%s|%s|%s\n", l.levels[v], l.prefix, format)
			fmt.Fprintf(l.out, f, args...)
		}
	}
}
