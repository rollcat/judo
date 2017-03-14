package libjudo

import (
	"fmt"
	"os"
	"path"
)

// Represents a single host (invocation target)
type Host struct {
	Name   string
	groups []string
	cancel chan bool
	master *Proc
}

func NewHost(name string) (host *Host) {
	return &Host{
		Name:   name,
		groups: []string{},
		cancel: make(chan bool),
		master: nil,
	}
}

func (host Host) Log(msg string) {
	logger.Printf("%s: %s\n", host.Name, msg)
}

func (host *Host) SendRemoteAndRun(job *Job) (err error) {
	// speedify!
	host.StartMaster()

	// deferred functions are called first in, last out.
	// any other defers can still use the master to clean up remote.
	defer host.StopMaster()

	// make cozy
	tmpdir, err := host.SshRead(job, "mktemp", "-d")
	if err != nil {
		return err
	}

	// ensure cleanup
	defer func() {
		if err := recover(); err != nil {
			// oops! clean up remote
			assert(host.Ssh(job, "rm", "-r", tmpdir))
			// continue panicking
			panic(err)
		}
	}()

	// push files to remote
	host.pushFiles(job, job.Script.fname, tmpdir)

	// are we in dirmode?
	var remote_command string
	if !job.Script.dirmode {
		remote_command = path.Join(tmpdir, path.Base(job.Script.fname))
	} else {
		remote_command = path.Join(
			tmpdir,
			path.Base(job.Script.fname),
			"script",
		)
	}

	// do the actual work
	err_job := host.Ssh(
		job,
		"cd", tmpdir, ";",
		"env",
		fmt.Sprintf("HOSTNAME=%s", host.Name),
		remote_command,
	)

	// clean up
	if err = host.Ssh(job, "rm", "-r", tmpdir); err != nil {
		return err
	}
	return err_job
}

func (host *Host) RunRemote(job *Job) (err error) {
	return host.Ssh(job, job.Command.cmd)
}

func (host *Host) Cancel() {
	go func() {
		// kill up to two: master and currently running
		host.cancel <- true
		host.cancel <- true
	}()
}

func (host *Host) StartMaster() (err error) {
	if host.master != nil {
		panic("there already is a master")
	}
	proc, err := NewProc("ssh", "-M", host.Name)
	if err != nil {
		return
	}
	host.master = proc
	go func() {
		close(host.master.Stdin())
		for host.master != nil {
			select {
			case line, ok := <-host.master.Stdout():
				if !ok {
					continue
				}
				host.Log(line)
			case line, ok := <-host.master.Stderr():
				if !ok {
					continue
				}
				host.Log(line)
			case err = <-host.master.Done():
				if err != nil {
					host.Log(err.Error())
				}
				host.master = nil
			case <-host.cancel:
				host.StopMaster()
				host.master = nil
			}
		}
	}()
	return
}

func (host *Host) StopMaster() error {
	if host.master == nil {
		panic("there was no master to stop")
	}
	return host.master.Signal(os.Interrupt)
}
