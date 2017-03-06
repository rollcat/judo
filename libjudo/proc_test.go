package libjudo

import (
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
