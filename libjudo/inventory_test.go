package libjudo

import (
	"strings"
	"testing"
)

func TestInventory_resolveNames(t *testing.T) {
	inventory := NewInventory([]string{})
	for host := range inventory.resolveNames("test") {
		if host.Name != "test" {
			t.Error("no host")
			return
		}
	}
}

func TestInventoryPopulate1(t *testing.T) {
	inventory := NewInventory([]string{"test"})
	inventory.Populate()
	host, ok := <-inventory.GetHosts()
	if !ok {
		t.Error("no host")
		return
	}
	if host.Name != "test" {
		t.Error("bad host")
		return
	}
}

func TestInventoryPopulate2(t *testing.T) {
	inventory := NewInventory([]string{"test1", "test2"})
	inventory.Populate()

	seen := make(map[string]bool)
	for host := range inventory.GetHosts() {
		seen[host.Name] = true
	}
	for _, name := range []string{"test1", "test2"} {
		if !seen[name] {
			t.Error("no host:", name)
			return
		}
	}
}

func TestExpandHostGroups_NoDuplicates(t *testing.T) {
	inventory := NewInventory([]string{"test1", "test2", "test1"})
	inventory.Populate()

	seen := make(map[string]bool)
	for host := range inventory.GetHosts() {
		if seen[host.Name] {
			t.Error("seen twice:", host.Name)
			return
		}
		seen[host.Name] = true
	}
}

func TestReadGroups(t *testing.T) {
	r := strings.NewReader("test1\ntest2\n")
	seen := make(map[string]bool)
	for _, name := range readGroups(r) {
		seen[name] = true
	}
	for _, name := range []string{"test1", "test2"} {
		if !seen[name] {
			t.Error("no host:", name)
			return
		}
	}
}

func TestReadGroupsComments(t *testing.T) {
	r := strings.NewReader(`# a comment
test1 # another comment
test2 garbage
# test3
`)
	expect := map[string]bool{"test1": true, "test2": true}
	for _, name := range readGroups(r) {
		if !expect[name] {
			t.Error("unexpected:", name)
		}
	}
}
