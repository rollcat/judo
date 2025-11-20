package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
)

// Host represents a single host (invocation target)
type Host struct {
	Name    string
	Env     map[string]string
	SshArgs []string
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
		Name:    name,
		Env:     env,
		SshArgs: []string{},
		groups:  []string{},
		cancel:  make(chan bool),
		master:  nil,
		logger:  log.New(os.Stderr, fmt.Sprintf("%s: ", name), 0),
	}
}

// Run the script. The Filename must exist and be either a regular,
// executable file, or a "dirmode" style directory (with an executable
// file named "script" inside).
func (s *Script) Run(ctx context.Context, cfg *Config, host *Host) (err error) {
	dirmode := false
	stat, err := os.Stat(s.Filename)
	if err != nil {
		return err
	}
	// figure out if we should run in dirmode
	if stat.IsDir() {
		_, err = os.Stat(path.Join(s.Filename, "script"))
		if err != nil {
			return err
		}
		dirmode = true
	}

	// speedify!
	host.StartMaster()

	// deferred functions are called first in, last out.
	// any other defers can still use the master to clean up remote.
	defer host.StopMaster()

	// make cozy
	err = host.SSH(cfg, `mkdir -p "$HOME/.judo"`)
	workdir, err := host.SSHRead(cfg, `TMPDIR="$HOME/.judo" mktemp -d`)
	if err != nil {
		return err
	}
	host.workdir = workdir

	cleanup := func() error {
		host.workdir = ""
		return host.SSH(cfg, fmt.Sprintf("rm -r %s", shquote(workdir)))
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
	if dirmode {
		remoteScriptDir = path.Join(
			workdir,
			s.Filename,
		)
	} else {
		remoteScriptDir = path.Join(
			workdir,
			path.Dir(s.Filename),
		)
	}

	// Create remote directory structure
	if err = host.SSH(
		cfg,
		fmt.Sprintf("mkdir -p %s", shquote(remoteScriptDir)),
	); err != nil {
		return err
	}

	// push files to remote
	if dirmode {
		if err = host.pushFiles(
			cfg,
			s.Filename,
			// We need to strip one level of path, otherwise
			// scp will duplicate it. Eh...
			path.Dir(remoteScriptDir),
		); err != nil {
			return err
		}
	} else {
		if err = host.pushFiles(
			cfg,
			s.Filename,
			remoteScriptDir,
		); err != nil {
			return err
		}
	}

	// Figure out what is the remote command
	var remoteCommand string
	if dirmode {
		remoteCommand = path.Join(remoteScriptDir, "script")
	} else {
		remoteCommand = path.Join(
			remoteScriptDir,
			path.Base(s.Filename),
		)
	}

	// do the actual work
	err = host.SSH(cfg, remoteCommand)

	// clean up
	if errCleanup := cleanup(); errCleanup != nil {
		return errCleanup
	}
	return err
}

// Cancel execution of code on the remote end.
func (host *Host) Cancel() {
	go func() {
		// kill up to two: master and currently running
		host.cancel <- true
		host.cancel <- true
	}()
}
