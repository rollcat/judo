package libjudo

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

type Proc struct {
	stdin  chan string
	stdout chan string
	stderr chan string
	done   chan error

	cmd *exec.Cmd
}

func (proc *Proc) Done() <-chan error {
	return proc.done
}

func (proc *Proc) Stdin() chan<- string {
	return proc.stdin
}

func (proc *Proc) Stdout() <-chan string {
	return proc.stdout
}

func (proc *Proc) Stderr() <-chan string {
	return proc.stderr
}

func (proc *Proc) CloseStdin() {
	close(proc.stdin)
}

func scanLines(r io.ReadCloser, ch chan<- string, done chan<- interface{}) {
	defer func() {
		r.Close()
		close(ch)
		done <- nil
	}()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		ch <- text
	}
	assert(scanner.Err())
}

func writeLines(w io.WriteCloser, ch <-chan string, done chan<- interface{}) {
	defer func() {
		w.Close()
		done <- nil
	}()
	for line := range ch {
		bytes := append([]byte(line), []byte("\n")[0])
		_, err := w.Write(bytes)
		assert(err)
	}
}

func NewProc(name string, args ...string) (proc *Proc, err error) {
	bufsz := 0
	proc = &Proc{
		stdin:  make(chan string, bufsz),
		stdout: make(chan string, bufsz),
		stderr: make(chan string, bufsz),
		done:   make(chan error),
		cmd:    exec.Command(name, args...),
	}
	done := make(chan interface{})
	pw0, err := proc.cmd.StdinPipe()
	assert(err)
	pr1, err := proc.cmd.StdoutPipe()
	assert(err)
	pr2, err := proc.cmd.StderrPipe()
	assert(err)
	go func() {
		assert(proc.cmd.Start())
		<-done
		<-done
		<-done
		err = proc.cmd.Wait()
		proc.done <- err
		close(proc.done)
	}()
	go writeLines(pw0, proc.stdin, done)
	go scanLines(pr1, proc.stdout, done)
	go scanLines(pr2, proc.stderr, done)
	return
}

func GetOutputLines(name string, args ...string) (<-chan string, error) {
	ch := make(chan string)
	proc, err := NewProc(name, args...)
	if err != nil {
		return nil, err
	}
	close(proc.Stdin())
	go func() {
		for {
			select {
			case line := <-proc.Stdout():
				ch <- line
			case <-proc.Stderr():
			case err := <-proc.Done():
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
