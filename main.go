package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
    [-d] [-f n] [-t s]`

const version = "judo 0.1-dev"

func ParseArgs(args []string) (
	job *libjudo.Job, msg string,
	status int, err error) {

	args, opts, err := getopt.GetOpt(args, "hvs:c:t:", nil)
	if err != nil {
		return nil, usage, 111, err
	}

	var script *libjudo.Script
	var command *libjudo.Command
	var timeout uint64 = 30

	for _, opt := range opts {
		switch opt.Opt() {
		case "-h":
			return nil, usage, 0, nil
		case "-v":
			return nil, version, 0, nil
		case "-s":
			script, err = libjudo.NewScript(opt.Arg())
			if err != nil {
				return nil, err.Error(), 111, nil
			}
		case "-c":
			command = libjudo.NewCommand(opt.Arg())
		case "-t":
			timeout, err = strconv.ParseUint(opt.Arg(), 10, 64)
			if err != nil {
				return nil, usage, 111, err
			}
			// case "-f":
			// 	forks, err = strconv.ParseUint(opt.Arg(), 10, 8)
			// 	if err != nil {
			// 		return nil, usage, 111, err
			// 	}
		}
	}

	if script == nil && command == nil {
		return nil, usage, 111, nil
	}

	inventory := libjudo.NewInventory()
	inventory.Timeout = time.Duration(timeout) * time.Second
	inventory.Populate(args)
	job = libjudo.NewJob(inventory, script, command, timeout)

	return job, "", 0, nil
}

func main() {
	job, msg, status, err := ParseArgs(os.Args[1:])
	job.InstallSignalHandlers()
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
	result := job.Execute()
	successful, failful := result.Report()
	if failful > 0 {
		if successful == 0 {
			os.Exit(2)
		} else {
			os.Exit(1)
		}
	}
	os.Exit(0)
}
