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
		stdin:  make(chan string),
		stdout: make(chan string),
		stderr: make(chan string),
		done:   make(chan error),
		cmd:    exec.Command(name, args...),
	}
	pw0, err := proc.cmd.StdinPipe()
	assert(err)
	pr1, err := proc.cmd.StdoutPipe()
	assert(err)
	pr2, err := proc.cmd.StderrPipe()
	assert(err)
	go writeLines(pw0, proc.stdin)
	go scanLines(pr1, proc.stdout)
	go scanLines(pr2, proc.stderr)
	assert(proc.cmd.Start())
	go func() {
		proc.done <- proc.cmd.Wait()
	}()
	return
}

func GetOutputLines(name string, args ...string) (ch chan string, err error) {
	ch = make(chan string)
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
