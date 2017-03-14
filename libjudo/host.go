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
	err = host.Ssh(job, "mkdir", "-p", "$HOME/.judo")
	tmpdir, err := host.SshRead(job, "TMPDIR=$HOME/.judo", "mktemp", "-d")
	if err != nil {
		return err
	}

	cleanup := func() error {
		return host.Ssh(job, "rm", "-r", tmpdir)
	}

	// ensure cleanup
	defer func() {
		if err := recover(); err != nil {
			// oops! clean up remote
			assert(cleanup())
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
	if err = cleanup(); err != nil {
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
	proc, err := NewProc("ssh", "-M", host.Name, "cat", "-")
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
				host.master.CloseStdin()
				host.StopMaster()
			}
		}
	}()
	return
}

func (host *Host) StopMaster() error {
	if host.master == nil || !host.master.IsAlive() {
		host.Log("there was no master to stop")
		return nil
	}
	return host.master.Signal(os.Interrupt)
}
