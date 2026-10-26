package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	getopt "github.com/rollcat/getopt"
)

const (
	errUsage = `judo: use "judo -h" to get help"`
	longHelp = `usage:
    judo [common flags] -s SCRIPT  [--] ssh-targets
    judo [common flags] -c COMMAND [--] ssh-targets
    judo -v [REQUIRED-VERSION]
    judo -h
common flags:  [-t TIMEOUT] [-e KEY | KEY=VALUE] [-F SSH_CONFIG] [-d]
flags:
    -s  Execute specified SCRIPT (file) on remote targets
    -c  Execute specified shell COMMAND on remote targets
    -v  Display the software version; check that this binary
        is backward compatible with REQUIRED-VERSION
    -h  Display this help text
    -t  Wait TIMEOUT seconds before giving up
    -e  Set KEY to VALUE in the remote environment
        (default: take the value from the local environment)
    -V  Instruct ssh(1)/scp(1) to be verbose (ssh -v)
    -F  Instruct ssh(1)/scp(1) to use custom SSH_CONFIG file
    -d  More verbose debugging logs`
)

const version = "0.6"

func parseArgs(args []string) (
	cfg *Config,
	names []string,
	msg string,
	status int,
	err error,
) {
	names, opts, err := getopt.GetOpt(args, "s:c:vht:e:F:d", nil)
	if err != nil {
		return nil, nil, errUsage, 111, err
	}

	steps := []Step{}
	timeout := time.Duration(30) * time.Second
	sshArgs := []string{}

	for _, opt := range opts {
		switch opt.Opt() {
		case "-e":
			env, err := parseEnvArg(opt.Arg())
			if err != nil {
				return nil, nil, errUsage, 111, err
			}
			steps = append(steps, env)
		case "-s":
			script := &Script{Filename: opt.Arg()}
			steps = append(steps, script)
		case "-c":
			command := &Command{opt.Arg()}
			steps = append(steps, command)
		case "-v":
			if len(names) > 0 && version != names[0] {
				return nil, nil, version, 1, nil
			}
			return nil, nil, version, 0, nil
		case "-V":
			sshArgs = append(sshArgs, "-v")
		case "-h":
			return nil, nil, longHelp, 0, nil
		case "-t":
			timeout, err = time.ParseDuration(opt.Arg())
			if err != nil {
				return nil, nil, errUsage, 111, err
			}
		case "-F":
			sshArgs = append(sshArgs, "-F", opt.Arg())
		case "-d":
			moreDebugLogging()
		default:
			panic("unexpected argument")
		}
	}

	if len(steps) == 0 {
		// errMsg := "error: no steps specified"
		return nil, nil, errUsage, 1, nil
	}

	for _, name := range names {
		if strings.Contains(name, "@") {
			errMsg := fmt.Sprintf("error: malformed hostname: %s", name)
			return nil, nil, errMsg, 1, nil
		}
	}

	inventory := NewInventory()
	inventory.Timeout = timeout
	cfg = NewConfig(inventory, steps, sshArgs)

	return cfg, names, "", 0, nil
}

type argumentError struct {
	Message string
}

func (e argumentError) Error() string {
	return fmt.Sprintf("Bad argument: %s", e.Message)
}

func parseEnvArg(arg string) (*Env, error) {
	elems := strings.SplitN(arg, "=", 2)
	key := elems[0]
	var value string
	switch len(elems) {
	case 0:
		panic("wtf")
	case 1:
		var ok bool
		if value, ok = os.LookupEnv(key); !ok {
			return nil, argumentError{
				Message: fmt.Sprintf("%s not set", key),
			}
		}
	case 2:
		value = elems[1]
	}
	return &Env{Name: key, Value: value}, nil
}

func main() {
	cfg, names, msg, status, err := parseArgs(os.Args[1:])
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
	cfg.PopulateInventory(names)
	cfg.InstallSignalHandlers()

	fmt.Printf("Running: %v\n", func() (names []string) {
		// look mama, Go has list comprehensions
		for host := range cfg.GetHosts() {
			names = append(names, host.Name)
		}
		return
	}())
	ctx := context.Background()
	cfg.Run(ctx)
	successful, failful := cfg.Report()
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
