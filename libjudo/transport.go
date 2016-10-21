package libjudo

import (
	"fmt"
	"time"
)

type TimeoutError struct {
}

func (e TimeoutError) Error() string {
	return "Operation timed out"
}

func (host *Host) pushFiles(fname_local string, fname_remote string) (err error) {
	var remote = fmt.Sprintf("[%s]:%s", host.Name, fname_remote)
	proc, err := NewProc("scp", "-r", fname_local, remote)
	if err != nil {
		return
	}
	close(proc.Stdin)
	for {
		select {
		case line := <-proc.Stdout:
			host.Log(line)
		case line := <-proc.Stderr:
			host.Log(line)
		case err = <-proc.Done:
			return err
		case <-time.After(10 * time.Second):
			return TimeoutError{}
		}
	}
}

func (host *Host) Ssh(command string, args ...string) (err error) {
	proc, err := NewProc(
		"ssh",
		append([]string{host.Name, command}, args...)...,
	)
	if err != nil {
		return
	}
	close(proc.Stdin)
	for {
		select {
		case line := <-proc.Stdout:
			host.Log(line)
		case line := <-proc.Stderr:
			host.Log(line)
		case err = <-proc.Done:
			return err
		case <-time.After(20 * time.Second):
			return TimeoutError{}
		}
	}
}

func (host *Host) SshRead(command string, args ...string) (out string, err error) {
	proc, err := NewProc(
		"ssh",
		append([]string{host.Name, command}, args...)...,
	)
	if err != nil {
		return
	}
	close(proc.Stdin)
	for {
		select {
		case line := <-proc.Stdout:
			out = line
		case line := <-proc.Stderr:
			host.Log(line)
		case err = <-proc.Done:
			return
		case <-time.After(10 * time.Second):
			return "", TimeoutError{}
		}
	}
}
