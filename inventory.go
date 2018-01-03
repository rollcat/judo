package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"time"
)

var inventoryLine = regexp.MustCompile("^[^# ]+")

// Inventory is a collection of managed hosts.
type Inventory struct {
	hosts   []*Host
	s       *SeenString
	Timeout time.Duration
	logger  Logger
}

// NewInventory creates a new Inventory.
func NewInventory() *Inventory {
	return &Inventory{
		hosts:   []*Host{},
		s:       NewSeenString(),
		Timeout: time.Duration(30) * time.Second,
		logger:  log.New(os.Stderr, "inventory: ", 0),
	}
}

// Populate the inventory with given names. Each name is resolved -
// the process consists of looking up potential group and host names;
// e.g. if you have a group named "foo" with hosts "a" and "b" in it,
// the inventory will be populated with hosts "a" and "b". If the
// hosts already exist in the inventory, they will be updated to
// reflect group membership.
func (inventory *Inventory) Populate(names []string) {
	for _, name := range names {
		for host := range inventory.resolveNames(name) {
			inventory.hosts = append(inventory.hosts, host)
		}
	}
}

// GetHosts iterates over all hosts in the inventory.
func (inventory *Inventory) GetHosts() (ch chan *Host) {
	ch = make(chan *Host)
	go func() {
		for _, host := range inventory.hosts {
			ch <- host
		}
		close(ch)
	}()
	return
}

func isExecutable(mode os.FileMode) bool {
	return (mode.Perm() & 0111) > 0
}

func readGroups(r io.Reader) (out []string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		name := inventoryLine.FindString(line)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	assert(scanner.Err())
	return
}

func (inventory *Inventory) readGroupsFromScript(fname string, ch chan *Host) {
	proc, err := NewProc(fname)
	assert(err)
	close(proc.Stdin())
	for {
		select {
		case line, ok := <-proc.Stdout():
			name := inventoryLine.FindString(line)
			if !ok || name == "" {
				continue
			}
			for host := range inventory.resolveNames(name) {
				ch <- host
			}
		case line, ok := <-proc.Stderr():
			if !ok {
				continue
			}
			inventory.logger.Print(line)
		case err = <-proc.Done():
			assert(err)
			return
		case <-time.After(inventory.Timeout):
			panic(ErrorTimeout)
		}
	}
}

func (inventory *Inventory) readGroupsFromFile(fname string, ch chan *Host) {
	f, err := os.Open(fname)
	assert(err)
	defer f.Close()
	for _, name := range readGroups(f) {
		for host := range inventory.resolveNames(name) {
			ch <- host
		}
	}
}

func (inventory *Inventory) resolveNames(name string) (ch chan *Host) {
	ch = make(chan *Host)
	fname := path.Join("groups", name)
	stat, err := os.Stat(fname)

	if err != nil {
		go func() {
			if !inventory.s.SeenBefore(name) {
				ch <- NewHost(name)
			}
			close(ch)
		}()
		return
	}

	if !stat.Mode().IsRegular() {
		close(ch)
		panic("not regular file")
	}
	go func() {
		defer close(ch)
		if isExecutable(stat.Mode()) {
			inventory.readGroupsFromScript(fname, ch)
		} else {
			inventory.readGroupsFromFile(fname, ch)
		}

	}()
	return
}
