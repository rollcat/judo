package libjudo

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
