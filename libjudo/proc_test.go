package libjudo

import (
	"fmt"
	"testing"
	"time"
)

func TestCat(t *testing.T) {
	proc, err := NewProc("cat", "-")
	if err != nil {
		t.Error(err)
		return
	}
	proc.Stdin() <- "hello"
	close(proc.Stdin())
	for {
		select {
		case line := <-proc.Stdout():
			if line != "hello" {
				t.Error("unexpected line", line)
			}
			return
		case err := <-proc.Done():
			if err != nil {
				t.Error(err)
				return
			}
			return
		case <-time.After(1 * time.Second):
			t.Error("timeout")
			return
		}
	}
}

func TestManyHellos(t *testing.T) {
	proc, err := NewProc("cat", "-")
	if err != nil {
		t.Error(err)
		return
	}
	hellos := 1 << 8
	for i := 0; i < hellos; i++ {
		select {
		case proc.Stdin() <- fmt.Sprintf("hello %d", i):
		case <-time.After(1 * time.Second / 100):
			t.Error("timeout stdin")
			return
		}
	}
	proc.CloseStdin()
	for {
		select {
		case line, ok := <-proc.Stdout():
			if !ok && line == "" {
				continue
			}
			if line[0:5] != "hello" {
				t.Error("unexpected line", line)
				return
			}
			hellos--
		case <-proc.Stderr():
		case err := <-proc.Done():
			if err != nil {
				t.Error(err)
				return
			}
			if hellos > 0 {
				t.Error("did not receive enough hellos:", hellos)
			} else if hellos < 0 {
				t.Error("too many hellos:", hellos)
			}
			return
		case <-time.After(1 * time.Second / 100):
			t.Error("timeout stdout/stderr")
			return
		}
	}
}

func TestEcho(t *testing.T) {
	ch, err := GetOutputLines("echo", "hello")
	if err != nil {
		t.Error(err)
		return
	}
	for line := range ch {
		if line != "hello" {
			t.Error("unexpected line", line)
		}
		return
	}
}
