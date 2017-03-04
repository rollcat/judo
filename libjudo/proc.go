package libjudo

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

type Proc struct {
	Stdin  chan string
	Stdout chan string
	Stderr chan string
	Done   chan error

	cmd *exec.Cmd
}

func scanLines(r io.ReadCloser, ch chan<- string) {
	defer close(ch)
	defer r.Close()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		assert(scanner.Err())
		ch <- scanner.Text()
	}
}

func writeLines(w io.WriteCloser, ch <-chan string) {
	defer w.Close()
	for line := range ch {
		_, err := w.Write([]byte(line))
		assert(err)
	}
}

func NewProc(name string, args ...string) (proc *Proc, err error) {
	proc = &Proc{
		Stdin:  make(chan string),
		Stdout: make(chan string),
		Stderr: make(chan string),
		Done:   make(chan error),
		cmd:    exec.Command(name, args...),
	}
	pr, pw := io.Pipe()
	proc.cmd.Stdout = pw
	go scanLines(pr, proc.Stdout)
	pr, pw = io.Pipe()
	proc.cmd.Stderr = pw
	go scanLines(pr, proc.Stderr)
	pr, pw = io.Pipe()
	proc.cmd.Stdin = pr
	go writeLines(pw, proc.Stdin)
	assert(proc.cmd.Start())
	go func() {
		proc.Done <- proc.cmd.Wait()
	}()
	return
}

func GetOutputLines(name string, args ...string) (ch chan string, err error) {
	ch = make(chan string)
	proc, err := NewProc(name, args...)
	if err != nil {
		return nil, err
	}
	close(proc.Stdin)
	go func() {
		for {
			select {
			case line := <-proc.Stdout:
				ch <- line
			case <-proc.Stderr:
			case err := <-proc.Done:
				close(ch)
				assert(err)
			}
		}
	}()
	return ch, nil
}

func (proc Proc) Signal(sig os.Signal) error {
	return proc.cmd.Process.Signal(sig)
}
