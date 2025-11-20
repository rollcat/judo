package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// Config represents the context and the main entry point to the
// program. It holds the host inventory and the steps to run on each
// host.
type Config struct {
	*Inventory
	Steps   []Step
	SshArgs []string
	Results map[*Host]error
	m       *sync.Mutex
	env     map[string]string
	signals chan os.Signal
}

// Step is a step/action to be run on a [Host]. The [context.Context]
// allows cancellation, while [Config] can be used to modify / consult
// the environment, and report results.
type Step interface {
	Run(context.Context, *Config, *Host) error
}

// Script represents the file/directory to be sent to the remote Host
// for execution.
type Script struct {
	Filename string
}

// Command represents an ad-hoc command to be executed on the remote
// Host.
type Command struct {
	CmdLine string
}

// Env holds an environment variable to add to the execution [Config]
// of each [Host].
type Env struct {
	Name, Value string
}

var (
	_ Step = &Script{}
	_ Step = &Command{}
	_ Step = &Env{}
)

// Report groups the result into successful and failed host names.
func (cfg *Config) Report() (successful []string, failful map[string]error) {
	failful = make(map[string]error)
	for host, err := range cfg.Results {
		if err == nil {
			successful = append(successful, host.Name)
		} else {
			failful[host.Name] = err
		}
	}
	return successful, failful
}

// Add the environment variable to the config.
func (e *Env) Run(_ context.Context, cfg *Config, _ *Host) error {
	cfg.env[e.Name] = e.Value
	return nil
}

// Run the command on the remote host.
func (c *Command) Run(_ context.Context, cfg *Config, host *Host) error {
	_, err := host.SSHRead(cfg, c.CmdLine)
	return err
}

// NewConfig prepares a new [Config].
func NewConfig(
	inventory *Inventory,
	steps []Step,
	sshArgs []string,
) *Config {
	// https://golang.org/pkg/os/signal/#Notify
	signals := make(chan os.Signal, 1)
	return &Config{
		Inventory: inventory,
		Steps:     steps,
		SshArgs:   sshArgs,
		Results:   map[*Host]error{},
		m:         &sync.Mutex{},
		env:       map[string]string{},
		signals:   signals,
	}
}

// InstallSignalHandlers installs a signal handler, which will catch
// interrupt requests, and cancel pending steps.
func (cfg Config) InstallSignalHandlers() {
	signal.Notify(cfg.signals, os.Interrupt)
	go func() {
		// wait for SIGINT
		<-cfg.signals
		// let everyone know we're cancelling the operation
		for host := range cfg.GetHosts() {
			host.Cancel()
		}
	}()
}

// PopulateInventory with given names; resolve additional arguments
// and environment overrides.
func (cfg Config) PopulateInventory(names []string) {
	cfg.Inventory.Populate(names)
	for host := range cfg.GetHosts() {
		host.SshArgs = cfg.SshArgs
		for key, value := range cfg.env {
			if _, has := host.Env[key]; has {
				panic(fmt.Sprintf("Tried to override: %s", key))
			}
			host.Env[key] = value
		}
	}
}

// Run the defined steps.
func (cfg *Config) Run(ctx context.Context) {
	// The heart of judo, run each [Step] on remote [Host]s

	// Showtime
	for host := range cfg.GetHosts() {
		go func(host *Host) {
			for _, step := range cfg.Steps {
				err := step.Run(ctx, cfg, host)
				if err != nil {
					cfg.m.Lock()
					cfg.Results[host] = err
					cfg.m.Unlock()
					return
				}
			}
			// Everything went fine
			cfg.m.Lock()
			cfg.Results[host] = nil
			cfg.m.Unlock()
		}(host)
	}
}
