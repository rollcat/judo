package main

import (
	"fmt"
	"os"
	"time"
)

func (host *Host) pushFiles(job *Job,
	fname_local string, fname_remote string) (err error) {
	var remote = fmt.Sprintf("[%s]:%s", host.Name, fname_remote)
	proc, err := NewProc("scp", "-r", fname_local, remote)
	if err != nil {
		return
	}
	close(proc.Stdin())
	for {
		select {
		case line, ok := <-proc.Stdout():
			if !ok {
				continue
			}
			host.logger.Println(line)
		case line, ok := <-proc.Stderr():
			if !ok {
				continue
			}
			host.logger.Println(line)
		case err = <-proc.Done():
			return err
		case <-time.After(job.Timeout):
			return TimeoutError
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return CancelError
		}
	}
}

func shquote(s string) string {
	// TODO: quote literal inline '
	return fmt.Sprintf(`'%s'`, s)
}

func (host *Host) startSsh(job *Job, command string) (proc *Proc, err error) {
	ssh_args := []string{host.Name}
	ssh_args = append(ssh_args, []string{"cd", host.tmpdir, "&&"}...)
	ssh_args = append(ssh_args, []string{"env"}...)
	for key, value := range host.Env {
		ssh_args = append(ssh_args, fmt.Sprintf("%s=%s", key, value))
	}
	ssh_args = append(ssh_args, []string{"sh", "-c", shquote(command)}...)
	return NewProc("ssh", ssh_args...)
}

func (host *Host) Ssh(job *Job, command string) (err error) {
	proc, err := host.startSsh(job, command)
	close(proc.Stdin())
	for {
		select {
		case line, ok := <-proc.Stdout():
			if !ok {
				continue
			}
			host.logger.Println(line)
		case line, ok := <-proc.Stderr():
			if !ok {
				continue
			}
			host.logger.Println(line)
		case err = <-proc.Done():
			return err
		case <-time.After(job.Timeout):
			return TimeoutError
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return CancelError
		}
	}
}

func (host *Host) SshRead(job *Job, command string) (out string, err error) {
	proc, err := host.startSsh(job, command)
	close(proc.Stdin())
	for {
		select {
		case line, ok := <-proc.Stdout():
			if !ok {
				continue
			}
			out = line
		case line, ok := <-proc.Stderr():
			if !ok {
				continue
			}
			host.logger.Println(line)
		case err = <-proc.Done():
			return
		case <-time.After(job.Timeout):
			return "", TimeoutError
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return "", CancelError
		}
	}
}

func (host *Host) StartMaster() (err error) {
	if host.master != nil {
		panic("there already is a master")
	}
	proc, err := NewProc("ssh", "-MN", host.Name)
	if err != nil {
		return
	}
	host.master = proc
	go func() {
		for host.master != nil {
			select {
			case line, ok := <-host.master.Stdout():
				if !ok {
					continue
				}
				host.logger.Println(line)
			case line, ok := <-host.master.Stderr():
				if !ok {
					continue
				}
				host.logger.Println(line)
			case err = <-host.master.Done():
				if err != nil {
					host.logger.Println(err.Error())
				}
				host.master = nil
			case <-host.cancel:
				host.master.CloseStdin()
				host.StopMaster()
			}
		}
	}()
	return
}

func (host *Host) StopMaster() error {
	if host.master == nil || !host.master.IsAlive() {
		host.logger.Println("there was no master to stop")
		return nil
	}
	return host.master.Signal(os.Interrupt)
}
