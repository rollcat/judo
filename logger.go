package main

import (
	"log"
	"os"
	"sync"
)

// Logger encapsulates the basic functionality present in the "log"
// package, allowing pluggable implementations.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// NilLogger is a Logger that does nothing.
type NilLogger struct{}

// Print does nothing.
func (l *NilLogger) Print(v ...interface{}) {
}

// Printf does nothing.
func (l *NilLogger) Printf(format string, v ...interface{}) {
}

// Println does nothing.
func (l *NilLogger) Println(v ...interface{}) {
}

var debugLogger Logger = &NilLogger{}
var onceEnableDebugLogging *sync.Once = &sync.Once{}

func moreDebugLogging() {
	onceEnableDebugLogging.Do(func() {
		debugLogger = log.New(os.Stderr, "debug: ", 0)
	})
}
