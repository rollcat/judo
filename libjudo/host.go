package libjudo

import (
	"fmt"
	"path"
)

// Represents a single host (invocation target)
type Host struct {
	Name   string
	groups []string
}

func NewHost(name string) (host *Host) {
	return &Host{Name: name, groups: []string{}}
}

func NewHostGroups(name string, groups []string) (host *Host) {
	return &Host{Name: name, groups: groups}
}

func (host Host) Log(msg string) {
	logger.Printf("%s: %s\n", host.Name, msg)
}

func (host *Host) SendRemoteAndRun(script *Script) (err error) {
	// make cozy
	tmpdir, err := host.SshRead("mktemp", "-d")
	if err != nil {
		return err
	}

	// ensure cleanup
	defer func() {
		if err := recover(); err != nil {
			// oops! clean up remote
			assert(host.Ssh("rm", "-r", tmpdir))
			// continue panicking
			panic(err)
		}
	}()

	// push files to remote
	host.pushFiles(script.fname, tmpdir)

	// are we in dirmode?
	var remote_command string
	if !script.dirmode {
		remote_command = path.Join(tmpdir, path.Base(script.fname))
	} else {
		remote_command = path.Join(
			tmpdir,
			path.Base(script.fname),
			"script",
		)
	}

	// do the actual work
	err = host.Ssh(
		"cd", tmpdir, ";",
		"env",
		fmt.Sprintf("HOSTNAME=%s", host.Name),
		remote_command,
	)

	// clean up
	if err = host.Ssh("rm", "-r", tmpdir); err != nil {
		return err
	}
	return
}

func (host *Host) RunRemote(command *Command) (err error) {
	return host.Ssh(command.cmd)
}
