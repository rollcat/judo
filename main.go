package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rollcat/judo/libjudo"

	getopt "github.com/timtadh/getopt"
)

var debug_level = 0

const usage = `usage:
    judo [common flags] -s script  [--] ssh-targets
    judo [common flags] -c command [--] ssh-targets
    judo [-v | -h]
common flags:
    [-e KEY=VALUE] [-f n] [-t s]`

const version = "judo 0.2-dev"

func ParseArgs(args []string) (
	job *libjudo.Job, names []string, msg string,
	status int, err error) {

	names, opts, err := getopt.GetOpt(args, "hvs:c:t:e:", nil)
	if err != nil {
		return nil, nil, usage, 111, err
	}

	var script *libjudo.Script
	var command *libjudo.Command
	var timeout uint64 = 30
	env := make(map[string]string)

	for _, opt := range opts {
		switch opt.Opt() {
		case "-h":
			return nil, nil, usage, 0, nil
		case "-v":
			return nil, nil, version, 0, nil
		case "-s":
			script, err = libjudo.NewScript(opt.Arg())
			if err != nil {
				return nil, nil, err.Error(), 111, nil
			}
		case "-c":
			command = libjudo.NewCommand(opt.Arg())
		case "-t":
			timeout, err = strconv.ParseUint(opt.Arg(), 10, 64)
			if err != nil {
				return nil, nil, usage, 111, err
			}
		case "-f":
			_, err = strconv.ParseUint(opt.Arg(), 10, 8)
			if err != nil {
				return nil, nil, usage, 111, err
			}
		case "-e":
			err = ParseEnvArg(opt.Arg(), env)
			if err != nil {
				return nil, nil, usage, 111, err
			}
		}
	}

	if script == nil && command == nil {
		return nil, nil, usage, 111, nil
	}

	inventory := libjudo.NewInventory()
	inventory.Timeout = time.Duration(timeout) * time.Second
	job = libjudo.NewJob(inventory, script, command, env, timeout)

	return job, names, "", 0, nil
}

type ArgumentError struct {
	Message string
}

func (e ArgumentError) Error() string {
	return fmt.Sprintf("Bad argument: %s", e.Message)
}

func ParseEnvArg(arg string, env map[string]string) error {
	elems := strings.SplitN(arg, "=", 2)
	if len(elems) < 2 {
		return ArgumentError{Message: "missing = when parsing env arg"}
	}
	key, value := elems[0], elems[1]
	if _, has := env[key]; has {
		return ArgumentError{
			Message: fmt.Sprintf("%s already supplied", key),
		}
	}
	env[key] = value
	return nil
}

func main() {
	job, names, msg, status, err := ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
		os.Exit(111)
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
