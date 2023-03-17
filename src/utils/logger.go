package utils

import (
	"fmt"
	"log"
	"io"
	"os"
)

const (
	// Log levels
	INFO = iota
	ERROR
	DEBUG
)

type logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

func NewLogger(out io.Writer) *logger {
	if out == nil {
		out = os.Stdout
	}

	return &logger{
		infoLogger:  log.New(out, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(out, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(out, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *logger) SetOutput(w io.Writer) {
	l.infoLogger.SetOutput(w)
	l.errorLogger.SetOutput(w)
	l.debugLogger.SetOutput(w)
}

// LogBasedOnLvlf logs a message based on the log level passed in
//
// You can use this function to log a message with a format string
//
// However, please ensure that the 
// lvl passed in is valid (i.e. INFO, ERROR, or DEBUG), otherwise this function will panic
func (l *logger) LogBasedOnLvlf(lvl int, format string, args ...any) {
	switch lvl {
	case INFO:
		l.Infof(format, args...)
	case ERROR:
		l.Errorf(format, args...)
	case DEBUG:
		l.Debugf(format, args...)
	default:
		panic(
			fmt.Sprintf(
				"error %d: invalid log level %d passed to LogBasedOnLvl()",
				DEV_ERROR,
				lvl,
			),
		)
	}
}

// LogBasedOnLvl is a wrapper for LogBasedOnLvlf() that takes a string instead of a format string
//
// However, please ensure that the 
// lvl passed in is valid (i.e. INFO, ERROR, or DEBUG), otherwise this function will panic
func (l *logger) LogBasedOnLvl(lvl int, msg string) {
	l.LogBasedOnLvlf(lvl, msg)
}

func (l *logger) Debug(args ...any) {
	l.debugLogger.Println(args...)
}

func (l *logger) Debugf(format string, args ...any) {
	l.debugLogger.Printf(format, args...)
}

func (l *logger) Info(args ...any) {
	l.infoLogger.Println(args...)
}

func (l *logger) Infof(format string, args ...any) {
	l.infoLogger.Printf(format, args...)
}

func (l *logger) Error(args ...any) {
	l.errorLogger.Println(args...)
}

func (l *logger) Errorf(format string, args ...any) {
	l.errorLogger.Printf(format, args...)
}
