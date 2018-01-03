package main

import (
	"log"
	"os"
	"sync"
)

type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type NilLogger struct{}

func (l *NilLogger) Print(v ...interface{}) {
}

func (l *NilLogger) Printf(format string, v ...interface{}) {
}

func (l *NilLogger) Println(v ...interface{}) {
}

var debugLogger Logger = &NilLogger{}
var onceEnableDebugLogging *sync.Once = &sync.Once{}

func MoreDebugLogging() {
	onceEnableDebugLogging.Do(func() {
		debugLogger = log.New(os.Stderr, "debug: ", 0)
	})
}
