package main

import (
	"fmt"
	"log"
	"os"
	// "strconv"

	"judo/libjudo"

	getopt "github.com/timtadh/getopt"
)

var debug_level = 0

const usage = `usage:
    judo [-d] [-f n] -s script  [--] ssh-targets
    judo [-d] [-f n] -c command [--] ssh-targets
    judo [-v | -h]`

const version = "judo 0.1-dev"

func ParseArgs(args []string) (
	job *libjudo.Job, msg string,
	status int, err error) {

	args, opts, err := getopt.GetOpt(args, "hvs:c:", nil)
	if err != nil {
		return nil, usage, 111, err
	}

	var script *libjudo.Script
	var command *libjudo.Command

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
			// case "-f":
			// 	forks, err = strconv.ParseUint(opt.Arg(), 10, 8)
			// 	if err != nil {
			// 		return nil, usage, 111, nil
			// 	}
		}
	}

	if script == nil && command == nil {
		return nil, usage, 111, nil
	}

	inventory := libjudo.NewInventory(args)
	inventory.Populate()
	job = libjudo.NewJob(inventory, script, command)

	return job, "", 0, nil
}

func main() {
	job, msg, status, err := ParseArgs(os.Args[1:])
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
