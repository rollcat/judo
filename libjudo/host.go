package libjudo

import (
	"fmt"
	"path"
)

// Represents a single host (invocation target)
type Host struct {
	Name   string
	groups []string
	cancel chan bool
}

func NewHost(name string) (host *Host) {
	return &Host{
		Name:   name,
		groups: []string{},
		cancel: make(chan bool),
	}
}

func (host Host) Log(msg string) {
	logger.Printf("%s: %s\n", host.Name, msg)
}

func (host *Host) SendRemoteAndRun(job *Job) (err error) {
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
	host.cancel <- true
}
