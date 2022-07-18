package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	getopt "github.com/timtadh/getopt"
)

const errUsage = `judo: use "judo -h" to get help"`
const longHelp = `usage:
    judo [common flags] -s SCRIPT  [--] ssh-targets
    judo [common flags] -c COMMAND [--] ssh-targets
    judo -v [REQUIRED-VERSION]
    judo -h
common flags:  [-t TIMEOUT] [-e KEY | KEY=VALUE] [-d]
flags:
    -s  Execute specified SCRIPT (file) on remote targets
    -c  Execute specified shell COMMAND on remote targets
    -v  Display the software version; check that this binary
        is backward compatible with REQUIRED-VERSION
    -h  Display this help text
    -t  Wait TIMEOUT seconds before giving up
    -e  Set KEY to VALUE in the remote environment
        (default: take the value from the local environment)
    -d  More verbose debugging logs`

const version = "0.5"

func parseArgs(args []string) (
	job *Job, names []string, msg string,
	status int, err error) {

	names, opts, err := getopt.GetOpt(args, "s:c:vht:e:d", nil)
	if err != nil {
		return nil, nil, errUsage, 111, err
	}

	var script *Script
	var command *Command
	var timeout = time.Duration(30) * time.Second
	env := make(map[string]string)

	for _, opt := range opts {
		switch opt.Opt() {
		case "-s":
			script, err = NewScript(opt.Arg())
			if err != nil {
				return nil, nil, errUsage, 111, nil
			}
		case "-c":
			command = NewCommand(opt.Arg())
		case "-v":
			if len(names) > 0 && version != names[0] {
				return nil, nil, version, 1, nil
			}
			return nil, nil, version, 0, nil
		case "-h":
			return nil, nil, longHelp, 0, nil
		case "-t":
			timeout, err = time.ParseDuration(opt.Arg())
			if err != nil {
				return nil, nil, errUsage, 111, err
			}
		case "-e":
			err = parseEnvArg(opt.Arg(), env)
			if err != nil {
				return nil, nil, errUsage, 111, err
			}
		case "-d":
			moreDebugLogging()
		}
	}

	if script == nil && command == nil {
		return nil, nil, errUsage, 111, nil
	}

	inventory := NewInventory()
	inventory.Timeout = timeout
	job = NewJob(inventory, script, command, env, timeout)

	return job, names, "", 0, nil
}

type argumentError struct {
	Message string
}

func (e argumentError) Error() string {
	return fmt.Sprintf("Bad argument: %s", e.Message)
}

func parseEnvArg(arg string, env map[string]string) error {
	elems := strings.SplitN(arg, "=", 2)
	key := elems[0]
	var value string
	var has bool
	switch len(elems) {
	case 0:
		panic("wtf")
	case 1:
		value, has = os.LookupEnv(key)
		if !has {
			return argumentError{
				Message: fmt.Sprintf("%s not set", key),
			}
		}
	case 2:
		value = elems[1]
	}
	if _, has = env[key]; has {
	}
	env[key] = value
	return nil
}

func main() {
	job, names, msg, status, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		status = 111
	}
	if msg != "" {
		fmt.Println(msg)
		os.Exit(status)
	}
	if status != 0 {
		os.Exit(status)
	}
	job.PopulateInventory(names)
	job.InstallSignalHandlers()

	fmt.Printf("Running: %v\n", func() (names []string) {
		// look mama, Go has list comprehensions
		for host := range job.GetHosts() {
			names = append(names, host.Name)
		}
		return
	}())
	result := job.Execute()
	successful, failful := result.Report()
	if len(failful) > 0 {
		for host := range failful {
			fmt.Printf("Failed: %s: %s\n", host, failful[host])
		}
	}
	if len(successful) > 0 {
		fmt.Printf("Success: %v\n", successful)
	}

	if len(failful) > 0 {
		if len(successful) == 0 {
			os.Exit(2)
		} else {
			os.Exit(1)
		}
	}
	os.Exit(0)
}
