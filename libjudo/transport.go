package libjudo

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
