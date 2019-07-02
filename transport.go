package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	sshControlPath      = "~/.ssh/judo-control-%C"
	sshControlPathOpt   = "-o ControlPath=" + sshControlPath
	sshBatchOpt         = "-o BatchMode=yes"
	sshControlMasterOpt = "-o ControlMaster=no"
)

func (host *Host) pushFiles(job *Job,
	fnameLocal string, fnameRemote string) (err error) {
	var remote = fmt.Sprintf("[%s]:%s", host.Name, fnameRemote)
	proc, err := NewProc(
		"scp",
		sshBatchOpt,
		sshControlPathOpt,
		sshControlMasterOpt,
		"-r",
		fnameLocal,
		remote,
	)
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
			return ErrorTimeout
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return ErrorCancel
		}
	}
}

func shquote(s string) string {
	var b bytes.Buffer
	b.WriteRune('\'')
	for _, c := range s {
		if c == '\'' { // ASCII "'"
			b.WriteString("'\\''")
		} else {
			b.WriteRune(c)
		}
	}
	b.WriteRune('\'')
	return b.String()
}

// Serialize an array of strings into a string that, when passed to a
// shell, will again be interpreted as the same array of strings.
func shargs(ss []string) string {
	var qs []string
	for _, s := range ss {
		qs = append(qs, shquote(s))
	}
	return strings.Join(ss, " ")
}

func (host *Host) startSSH(job *Job, command string) (proc *Proc, err error) {
	sshArgs := []string{
		sshBatchOpt,
		sshControlPathOpt,
		sshControlMasterOpt,
		host.Name,
	}
	if host.workdir != "" {
		sshArgs = append(sshArgs, []string{
			"cd", host.workdir, "&&",
		}...)
	}
	sshArgs = append(sshArgs, []string{"env"}...)
	for key, value := range host.Env {
		sshArgs = append(sshArgs, fmt.Sprintf("%s=%s", key, value))
	}
	sshArgs = append(sshArgs, []string{"sh", "-c", shquote(command)}...)
	return NewProc("ssh", sshArgs...)
}

// SSH executes the given shell command on the remote host, and
// reports exist status.
func (host *Host) SSH(job *Job, command string) (err error) {
	proc, err := host.startSSH(job, command)
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
			return ErrorTimeout
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return ErrorCancel
		}
	}
}

// SSHRead executes the given shell command on the remote host, and
// returns its output together with exit status.
func (host *Host) SSHRead(job *Job, command string) (out string, err error) {
	proc, err := host.startSSH(job, command)
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
			return "", ErrorTimeout
		case <-host.cancel:
			if proc.IsAlive() {
				proc.Signal(os.Interrupt)
			}
			return "", ErrorCancel
		}
	}
}

// StartMaster starts the SSH master process for this host, to speed
// up execution of consecutive SSH requests.
func (host *Host) StartMaster() (err error) {
	if runtime.GOOS == "windows" {
		// Master process on Windows seems problematic
		return nil
	}
	if host.master != nil {
		panic("there already is a master")
	}
	proc, err := NewProc("ssh", sshBatchOpt, sshControlPathOpt, "-MN", host.Name)
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

// StopMaster kills the master process.
func (host *Host) StopMaster() error {
	if host.master == nil || !host.master.IsAlive() {
		host.logger.Println("there was no master to stop")
		return nil
	}
	return host.master.Signal(os.Interrupt)
}
