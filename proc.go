package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

// Proc wraps an OS process with a friendly, channel-based interface
// and line buffering.
type Proc struct {
	stdin  chan string
	stdout chan string
	stderr chan string
	done   chan error

	cmd *exec.Cmd
}

// Done returns a channel, on which the caller should wait for the
// process to complete. The channel will return an error status
// corresponding to the process' exit code. Only one caller can wait
// on this channel.
func (proc *Proc) Done() <-chan error {
	return proc.done
}

// Stdin returns a channel, which can be used to send input lines to
// the process. The channel is shared between callers.
func (proc *Proc) Stdin() chan<- string {
	return proc.stdin
}

// Stdout returns a channel, on which successive lines from process'
// stdout can be received. The channel is shared between callers.
func (proc *Proc) Stdout() <-chan string {
	return proc.stdout
}

// Stderr returns a channel, on which successive lines from process'
// stderr can be received. The channel is shared between callers.
func (proc *Proc) Stderr() <-chan string {
	return proc.stderr
}

// CloseStdin closes the standard input of the process.
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

// NewProc allocates and starts a new Proc.
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
		proc.cmd = nil
	}()
	go writeLines(pw0, proc.stdin, done)
	go scanLines(pr1, proc.stdout, done)
	go scanLines(pr2, proc.stderr, done)
	return
}

// IsAlive reports whether the process is still running.
func (proc Proc) IsAlive() bool {
	return proc.cmd != nil
}

// Signal sends the given signal to proc.
func (proc Proc) Signal(sig os.Signal) error {
	if !proc.IsAlive() {
		panic("process already dead")
	}
	return proc.cmd.Process.Signal(sig)
}
