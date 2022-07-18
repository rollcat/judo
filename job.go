package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"
)

// Script represents the file/directory to be sent to the remote Host
// for execution, potentially as a part of a Job.
type Script struct {
	fname   string
	dirmode bool
}

// Command represents an ad-hoc command to be executed on the remote
// Host, potentially as a part of a Job.
type Command struct {
	cmd string
}

// Job is a container for the host inventory and the script/commands
// to run on them.
type Job struct {
	*Inventory
	*Script
	*Command
	Timeout time.Duration
	AddEnv  map[string]string
	SshArgs []string
	signals chan os.Signal
}

// JobResult holds the per-host results of executing a Job.
type JobResult map[*Host]error

// Report groups the result into successful and failed host names.
func (result *JobResult) Report() (successful []string, failful map[string]error) {
	failful = make(map[string]error)
	for host := range *result {
		err := (*result)[host]
		if err == nil {
			successful = append(successful, host.Name)
		} else {
			failful[host.Name] = err
		}
	}
	return successful, failful
}

// NewCommand creates a Command.
func NewCommand(cmd string) (command *Command) {
	return &Command{cmd}
}

// NewScript creates a script. The named file/directory must exist and
// be either a regular, executable file, or a "dirmode" style
// directory (with an executable file named "script" inside).
func NewScript(fname string) (script *Script, err error) {
	script = &Script{fname: fname, dirmode: false}
	stat, err := os.Stat(script.fname)
	if err != nil {
		return nil, err
	}
	// figure out if we should run in dirmode
	if stat.IsDir() {
		stat, err = os.Stat(path.Join(script.fname, "script"))
		if err != nil {
			return nil, err
		}
		script.dirmode = true
	}
	return script, nil
}

// IsDirMode reports whether we're executing in dirmode.
func (script *Script) IsDirMode() bool {
	return script.dirmode
}

// NewJob creates a new Job object.
func NewJob(
	inventory *Inventory, script *Script, command *Command,
	env map[string]string, sshArgs []string,
	timeout time.Duration) (job *Job) {
	// https://golang.org/pkg/os/signal/#Notify
	signals := make(chan os.Signal, 1)
	return &Job{
		Inventory: inventory,
		Command:   command,
		Script:    script,
		Timeout:   timeout,
		AddEnv:    env,
		SshArgs:   sshArgs,
		signals:   signals,
	}
}

// InstallSignalHandlers installs a signal handler, which will catch
// interrupt requests, and cancel pending jobs.
func (job Job) InstallSignalHandlers() {
	signal.Notify(job.signals, os.Interrupt)
	go func() {
		// wait for SIGINT
		<-job.signals
		// let everyone know we're cancelling the operation
		for host := range job.GetHosts() {
			host.Cancel()
		}
	}()
}

// PopulateInventory with given names; resolve additional arguments
// and environment overrides.
func (job Job) PopulateInventory(names []string) {
	job.Inventory.Populate(names)
	for host := range job.GetHosts() {
		host.SshArgs = job.SshArgs
		for key, value := range job.AddEnv {
			if _, has := host.Env[key]; has {
				panic(fmt.Sprintf("Tried to override: %s", key))
			}
			host.Env[key] = value
		}
	}
}

// Execute is the entry point of a Job.
func (job *Job) Execute() *JobResult {
	// The heart of judo, run the Job on remote Hosts

	// Deliver the results of the job's execution on each Host
	var results = make(map[*Host]chan error)

	// Showtime
	for host := range job.GetHosts() {
		ch := make(chan error)
		results[host] = ch
		go func(host *Host, ch chan error) {
			var err error
			if job.Script != nil {
				err = host.SendRemoteAndRun(job)
			} else if job.Command != nil {
				err = host.RunRemote(job)
			} else {
				panic("Should not happen")
			}
			ch <- err
			close(ch)
		}(host, ch)
	}

	// Stats
	var jobresult JobResult = make(map[*Host]error)
	for host := range results {
		jobresult[host] = <-results[host]
	}
	return &jobresult
}
