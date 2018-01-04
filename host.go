package main

import (
	"fmt"
	"log"
	"os"
	"path"
)

// Host represents a single host (invocation target)
type Host struct {
	Name    string
	Env     map[string]string
	groups  []string
	workdir string
	cancel  chan bool
	master  *Proc
	logger  *log.Logger
}

// NewHost creates a new Host struct with default values.
func NewHost(name string) (host *Host) {
	env := make(map[string]string)
	env["HOSTNAME"] = name
	return &Host{
		Name:   name,
		Env:    env,
		groups: []string{},
		cancel: make(chan bool),
		master: nil,
		logger: log.New(os.Stderr, fmt.Sprintf("%s: ", name), 0),
	}
}

// SendRemoteAndRun establishes a connection to the host, sends off
// and executes the given job, and returns any possible resulting
// error.
func (host *Host) SendRemoteAndRun(job *Job) (err error) {
	// speedify!
	host.StartMaster()

	// deferred functions are called first in, last out.
	// any other defers can still use the master to clean up remote.
	defer host.StopMaster()

	// make cozy
	err = host.SSH(job, "mkdir -p $HOME/.judo")
	workdir, err := host.SSHRead(job, "TMPDIR=$HOME/.judo mktemp -d")
	if err != nil {
		return err
	}
	host.workdir = workdir

	cleanup := func() error {
		host.workdir = ""
		return host.SSH(job, fmt.Sprintf("rm -r %s", workdir))
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

	// Figure out the correct path for the remote script
	var remoteScriptDir string
	if job.Script.dirmode {
		remoteScriptDir = path.Join(
			workdir,
			job.Script.fname,
		)
	} else {
		remoteScriptDir = path.Join(
			workdir,
			path.Dir(job.Script.fname),
		)
	}

	// Create remote directory structure
	if err = host.SSH(
		job,
		fmt.Sprintf("mkdir -p %s", remoteScriptDir),
	); err != nil {
		return err
	}

	// push files to remote
	if job.Script.dirmode {
		if err = host.pushFiles(
			job,
			job.Script.fname,
			// We need to strip one level of path, otherwise
			// scp will duplicate it. Eh...
			path.Dir(remoteScriptDir),
		); err != nil {
			return err
		}
	} else {
		if err = host.pushFiles(
			job,
			job.Script.fname,
			remoteScriptDir,
		); err != nil {
			return err
		}
	}

	// Figure out what is the remote command
	var remoteCommand string
	if job.Script.dirmode {
		remoteCommand = path.Join(remoteScriptDir, "script")
	} else {
		remoteCommand = path.Join(
			remoteScriptDir,
			path.Base(job.Script.fname),
		)
	}

	// do the actual work
	errJob := host.SSH(job, remoteCommand)

	// clean up
	if err = cleanup(); err != nil {
		return err
	}
	return errJob
}

// RunRemote runs the given job on the host, assuming the connection
// has been already established, and job files copied over.
func (host *Host) RunRemote(job *Job) (err error) {
	return host.SSH(job, job.Command.cmd)
}

// Cancel execution of code on the remote end.
func (host *Host) Cancel() {
	go func() {
		// kill up to two: master and currently running
		host.cancel <- true
		host.cancel <- true
	}()
}
