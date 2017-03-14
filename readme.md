# Judo

Simple orchestration & configuration management.

Script common tasks in your favourite programming language.

Send them off to remote hosts for execution.

## Hello world

Let's create `hello.sh` with a `chmod +x`, and put this inside:

    #!/bin/sh
    set -eu
    echo "Hello from ${HOSTNAME}!"

Assuming `judo` is present in your `$PATH`, we can tell some remote
machine, called `george`, to execute `hello.sh`:

    judo -s hello.sh george

You will see the following output in your terminal:

    Running: [george]
    george: Hello from george!
    Success: [george]

If the task at hand doesn't facilitate writing a complete script, an
effect can be achieved by using the `-c` (for command) flag instead:

    judo -c "echo Hello again." george

    Running: [george]
    george: Hello again.
    Success: [george]

## Architecture

A typical setup will have a control machine and some remote machines.

The control machine can be anything moderately [UNIX][]-ish, capable
of running [Go][golang] executables and making outbound [SSH][]
connections: a dedicated machine, a VM in the cloud, a continuous
integration system, your own laptop, etc.

[golang]: https://golang.org/
[UNIX]: https://en.wikipedia.org/wiki/Unix
[SSH]: https://en.wikipedia.org/wiki/Secure_Shell

The remote machines will usually run some flavor of UNIX, but anything
that can be connected to over SSH can be made to obey to some extent
(via `-c`).

How much can be done, depends entirely on what's supported on the
remote host. There are no agents or clients or daemons; the remote
machine just receives your orders and executes them.

## Dependencies

The list of hard dependencies is intentionally kept minimal.

### Building

- A [Go][golang] compiler

### Control machine

- [`ssh(1)`][man-ssh]
- [`scp(1)`][man-ssh]
- A UNIX-flavored file system, that understands things like `chmod +x`

[man-ssh]: https://www.openssh.com/manual.html

### Remote machines

- [`env(1)`](https://linux.die.net/man/1/env) *
- [`mktemp(1)`](https://linux.die.net/man/1/mktemp) *
- [`rm(1)`](https://linux.die.net/man/1/rm) *
- [`sshd(8)`][man-ssh]

> \* consult your own operating system's manual pages!

## Installation

### On control machine

The real thing:

1. Get [Go][golang]
2. `go get github.com/rollcat/judo`

For development:

1. Get [Go][golang] anyway
2. Clone [the source](https://github.com/rollcat/judo)
3. `go build`

### On remote machines

No.

## Reaching the remote machines

### Using the SSH transport (without abusing it)

Judo will honor the standard `/etc/ssh/config` and `~/.ssh/config` to
establish & control the SSH connection. This is where you should keep
all of your dirty hacks like `ProxyCommand` or `Control*`.

### Groups: using with multiple remote hosts

So far, Judo might seem no more useful than this little tapeworm:

    scp foo.sh bar: && ssh bar "chmod +x foo.sh && ./foo.sh"

It can however execute commands on *groups* of hosts in parallel.

One way to achieve this, is to simply name several hosts on the
command line (e.g. `judo -s foo.sh fred george`). Another, more
scalable and manageable solution, is to create some sort of an
inventory, and refer to hosts by group names.

1. Create a directory named `groups`.

2. In there, create a file with the name of the group, e.g. `weasley`.

3. Put one hostname on each line in this file, like this:

        bill
        charlie
        fred
        george
        ginny
        percy
        ron

Congratulations, you now have a host inventory! Run this command to
see how it works in action:

    judo -s hello.sh weasley

Expect something similar to this:

    Running: [bill charlie fred george ginny percy ron]
    bill:    Hello from bill!
    charlie: Hello from charlie!
    george:  Hello from george!
    ginny:   Hello from ginny!
    percy:   Hello from percy!
    ron:     Hello from ron!
    Success: [bill charlie george ginny percy ron]
    Failed: fred: ssh: connect to host fred port 22: Connection timed out

Everyone but `fred` reported success.

Groups can be nested. It may be a good idea to create a group named
`all`, that will include all hosts and groups you need to manage.

The group file format permits blank lines, treats lines starting with
a `#` as comments, and ignores the remainder of a line if it finds a
space.

### Dynamic inventory

Sometimes you don't know the list of hosts ahead of time, or prefer to
keep it in another format or source. When Judo sees the `+x` flag on a
group file, it will execute it before starting the job; the output
(one line per host, like above) will indicate group membership.

You can use this to pull the list of hosts from some API, for example
when working with EC2:

    groups/
        ec2-eu-west-1
        ec2-eu-central-1
        ec2-all

Here, the files `ec2-eu-west-1` and `ec2-eu-central-1`  talk to
corresponding regions on EC2 to pull the host lists. The file
`ec2-all` is a simple text file, with two lines, naming each of the
above scripts.

Group scripts are not executed, unless the group name is included in
the job. So running `judo -s foo.sh fred` will not trigger any EC2 API
calls.

## Scripting

Writing and using scripts with Judo is extremely straightforward. You
probably already have some simple (or complex) administrative scripts,
these may be able to work completely unchanged.

Any executable (`chmod +x`) script can be used with `-s`. The only
requirement is that the remote host be able to locate the interpreter
(the `#!` line), and execute it. You can even send a precompiled
executable, or whatever else the target OS is capable of running.

### Simple example: report load average with Python

Create a file named `loadavg`:

    #!/usr/bin/env python
    import os
    print(os.getloadavg())

Judo wants the `+x` bit to be present before it will allow sending it
off to a host:

    chmod +x loadavg

We're ready to run it:

    judo -s loadavg all

Observe the results:

    Running: [armchair bed desk]
    armchair: (0.05, 0.17, 0.17)
    desk: (0.01, 0.02, 0.05)
    bed: (0.322265625, 0.15869140625, 0.09912109375)
    Success: [armchair bed desk]

Note there's no need for a file extension. Judo ignores it; it's the
`+x` bit and the hashbang line that matter. Use this to give your
scripts simple, readable names, and hide the implementation details.

### Complex / portable example: install Ruby on the remote host

Create a file named `install-ruby`:

```
#!/bin/sh
set -eu
case `uname` in
    Linux)
        case `lsb_release -si` in
            Debian)
                export DEBIAN_FRONTEND=noninteractive
                sudo -n apt-get install --yes ruby
                ;;
            *)
                echo >&2 "Unsupported Linux distro"
                exit 1
                ;;
        esac
        ;;
    OpenBSD)
        doas -n pkg_add -I ruby
        ;;
    *)
        echo >&2 "Unsupported operating system"
        exit 1
        ;;
esac
```

Install Ruby on all of our servers in the `eu-central-1` region:

    judo -s install-ruby ec2-eu-central-1

As you can see, while the operational model is straightforward, in
practice it can be a little tedious to write correct scripts that work
universally across heterogeneous environments.

This issue can be avoided either by keeping your boxes homogeneous, or
by using a library of OS-independent abstractions. Either way, this is
not Judo's problem.

### Passing arguments to scripts

No.

### Directories

Judo will also happily transfer an entire directory to the remote
host. In such case, it will first look for an executable file named
`script` inside, and run it on the remote host after the transfer of
the directory contents is complete.

The script can then refer to other files inside of that directory,
e.g. in order to copy configuration files to target destinations.

### Privilege escalation

No.

Judo in itself doesn't include any mechanisms to facilitate privilege
escalation. Use whatever means your remote machine provides.

You are probably already using `sudo` without a password.

If you're uneasy about this idea, consider that an encrypted private
SSH key is already a form of two-factor authentication (you have to
HAVE the key and KNOW the passphrase), and that if an intruder is able
to impersonate you on the target machine, you already have been owned.

### Input

No.

Standard input (file descriptor 0) is closed before the script runs.
Any attempts to read from `stdin` will immediately fail.

You should design your scripts, so that no input is required, other
than any environment variables provided, and to fail early and loudly
otherwise. Later, this will make it much easier to plug in Judo e.g.
into a continuous deployment pipeline.

Various options were considered to allow programs to read input, but
ultimately all of them suck. If you MUST work with a program that
DEMANDS interactive input, [`expect(1)`][expect] may be able to help
you.

[expect]: https://en.wikipedia.org/wiki/Expect

Programs that unexpectedly expect input, while used in an automated
context, often hang waiting for it for many minutes, before ultimately
timing out. It's best to `close(2)` the problem before it becomes a
real issue.

### Environment

Judo doesn't screw around with the remote script's environment. It
also tries to make the entire operation reliable, repeatable, and free
from unexpected side-effects; partly by not getting in your way, and
partly by enforcing some design principles.

You can always expect the following:

- For any `host`, running `ssh host env` will describe the environment
  fully, with one small difference, explained below.

- The environment variable `HOSTNAME` will be present, and will be set
  to the target's host name, as invoked on Judo's command line.

- Standard input will be closed, so if the remote machine tries to ask
  you something, it will only see an end-of-file.

- The current working directory will be a temporary target area, with
  the script file under the same relative path as on the control
  machine.

    For example, if you invoked Judo as `judo -s foo/script host`,
    then your file might exist in `~/.judo/tmp.rsDa2Er8ZC/foo/script`,
    where `~/.judo/tmp.rsDa2Er8ZC` will be your working directory.

    This is very useful when sending directories; `judo -s foo host`
    will transfer the entire `foo` directory to the remote, where
    `foo/script` could e.g. source an extra script, with a snippet
    similar to this one

        if [ -r /etc/centos-release ]; then
            . foo/vars_CentOS
        fi

### Check mode

No.

Use a staging environment. If you can't afford a staging environment,
use a local VM. If you can't afford a local VM, you seriously should
reconsider, why are you in system administration.

### Return value

Judo will return 0 if everything went all right, and something else if
something went wrong. Depending on what exactly went wrong, you can
expect these return codes:

- 1 if some hosts failed, but some succeeded
- 2 if all hosts failed
- 111 if there was an issue with how Judo was invoked (e.g. wrong
  flags, something wrong with a script, etc)

In any case (failure or not), look carefully at the output: it's meant
to be terse, but informative.

## Complete example

You should keep your stuff in revision control. [Git][git] is good for
this, [but][darcs] [pick][bzr] [whatever][svn] [you][cvs] [like][hg].

[bzr]: http://bazaar.canonical.com/
[cvs]: https://savannah.nongnu.org/projects/cvs
[darcs]: http://darcs.net/
[git]: https://git-scm.com/
[hg]: https://www.mercurial-scm.org/
[svn]: https://subversion.apache.org/

Here's an example directory hierarchy; directories have a trailing
"`/`", and executables have a "`*`":

```
readme.md                 - notes explaining the setup
Makefile                  - common tasks expressed with make
bin/                      - scripts to be run locally
    wait-for-host*        - keep pinging a host until it responds
groups/                   - inventory
    all                   - simply lists "ec2", "hetzner" and "home"
    ec2*                  - talks to EC2 API
    hetzner               - lists a few remote hosts
    servers               - lists "ec2", "hetzner", your home NAS, etc
    home                  - lists your workstation, router, NAS...
scripts/                  - collection of scripts to run on remotes
    bootstrap/            - get a new machine ready for action
        script*           - main entry point of the bootstrap job
        vars_CentOS       - OS-dependent vars,
        vars_Debian       - can be included with something like:
        vars_FreeBSD      - . scripts/bootstrap/vars_$(uname)
        vars_Linux        - and from vars_Linux:
        vars_OpenBSD      - . scripts/bootstrap/vars_$(lsb_release -si)
        vars_Raspbian     - but keep it simple!
    deploy-news.rollc.at* - deploy a web app
    deploy-www.rollc.at*  - deploy a simple blog
    install-docker*       - install some common packages
    install-nginx*        - useful for web hosting, etc
    update-system*        - or to update all installed packages
.git/                     - git cruft
```

For example, to update the contents of `www.rollc.at`, we could use:

    judo -s scripts/deploy-www.rollc.at chewie.rollc.at

Since this is a bit long and extremely common, there's a shortcut in
the `Makefile`:

    make deploy-www.rollc.at

Next time a kernel bug hits, installing the updated packages could be
as simple as:

    judo -s scripts/update-system servers

Followed by a reboot:

    judo -c reboot servers

If we're really anxious to see a particular server coming back up, we
could use one of our custom glue scripts:

    bin/wait-for-host -du chewie.rollc.at

## Using as a library (don't)

Most of the functionality is implemented internally as a library.
Many of the interfaces are public, but perhaps shouldn't be.
Don't use `libjudo` until 1.0 comes out!

## Known issues

### It's slow!

It's not significantly slower than running plain [SSH][].

If you are using session sharing (`ControlMaster` and friends) in auto
mode with `ControlPersist`, Judo will wait until the master in the
background exits. If it's not cool, you're advised to disable it.

You could also keep a master SSH connection open somewhere else in the
background. Since Judo invokes [`ssh(1)`][man-ssh] and
[`scp(1)`][man-ssh] a couple of times per each session, this may speed
things up significantly.

## A book?

If this readme does not describe everything that you need to know in
order to make Judo do useful work to you, then I have failed. Please
send suggestions on improving the readme.

## A bug?

Sure! Send me bug reports, suggestions, diffs, hate mail and
accolades - as appropriate. Contact info below.

Please do mind the following however:

- Judo is intentionally simple. It aims to have 90% of
  [Ansible's][ansible] core features, while having 1% of the code (or
  less). If Judo can't do something for you, you should investigate
  the following:

    - Rethink your problem,
    - Look into using [Ansible][ansible] instead.

[ansible]: https://www.ansible.com/

- Integrations with specific OS's on remote targets are left to
  separate projects - barring the minimum needed to set up the remote
  execution environment. However, Judo should make it simple to
  bootstrap some sort of "rosetta stone" on the remote end.

- The SLOC count of the entire source code (`grep -v '^$' $(find .
  -name "*.go") | wc -l`) shall never exceed 2000 lines.

- The dependencies, for building or for running, on control or on
  remote machines, shall never change.

- When the 1.0 release appears, the feature set and the interface will
  be frozen until it's time for a 2.0. All (well written) scripts must
  continue working, unchanged. If 2.0 ever appears, well written
  scripts should require no adjustments at all.

- The time for 2.0 will not be after 1.9; 1.9 will be followed by 1.10
  and 1.11, all the way to 1.61, if necessary. Same for 2.x; expect
  2.71 before 3.0.

## Author

&copy; 2016 Kamil Cholewiński <<kamil@rollc.at>>

License is [MIT](/license.txt).
