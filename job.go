package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"
)

// file/directory to be sent to the remote Host for execution
type Script struct {
	fname   string
	dirmode bool
}

// ad-hoc command to be executed on the remote Host
type Command struct {
	cmd string
}

// a set of Hosts on which to run Scripts/Commands
type Job struct {
	*Inventory
	*Script
	*Command
	Timeout time.Duration
	AddEnv  map[string]string
	signals chan os.Signal
}

// Holds the result of executing a Job
type JobResult map[*Host]error

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

func NewCommand(cmd string) (command *Command) {
	return &Command{cmd}
}

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

func (script *Script) IsDirMode() bool {
	return script.dirmode
}

func NewJob(
	inventory *Inventory, script *Script, command *Command,
	env map[string]string, timeout uint64) (job *Job) {
	// https://golang.org/pkg/os/signal/#Notify
	signals := make(chan os.Signal, 1)
	return &Job{
		Inventory: inventory,
		Command:   command,
		Script:    script,
		Timeout:   time.Duration(timeout) * time.Second,
		AddEnv:    env,
		signals:   signals,
	}
}

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

func (job Job) PopulateInventory(names []string) {
	job.Inventory.Populate(names)
	for host := range job.GetHosts() {
		for key, value := range job.AddEnv {
			if _, has := host.Env[key]; has {
				panic(fmt.Sprintf("Tried to override: %s", key))
			}
			host.Env[key] = value
		}
	}
}

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
