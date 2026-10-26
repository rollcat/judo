package main

import (
	"strings"
	"testing"
)

func TestMainParseHelp(t *testing.T) {
	_, _, msg, status, err := parseArgs([]string{})
	if err != nil {
		t.Error("err not nil")
	}
	if msg != errUsage {
		t.Error("no usage message")
	}
	if status == 0 {
		t.Error("no error status")
	}

	_, _, msg, status, err = parseArgs([]string{"-h"})
	if err != nil {
		t.Error("err not nil")
	}
	if msg != longHelp {
		t.Error("no help message")
	}
	if status != 0 {
		t.Error("unexpected error status")
	}

	_, _, msg, status, err = parseArgs([]string{"-v"})
	if err != nil {
		t.Error("err not nil")
	}
	if msg != version {
		t.Error("no version message")
	}
	if status != 0 {
		t.Error("unexpected error status")
	}
}

func TestMainParseCommand(t *testing.T) {
	cfg, _, _, _, err := parseArgs([]string{"-c", "true"})
	if cfg == nil {
		t.Error("cfg nil")
	}
	if err != nil {
		t.Error("err not nil")
	}
	if cfg.Steps == nil || len(cfg.Steps) == 0 {
		t.Error("no steps")
	}
	cmd := cfg.Steps[0]
	if cmd == nil {
		t.Error("cmd nil")
	}
}

func TestMainParseScript(t *testing.T) {
	cfg, _, _, _, err := parseArgs([]string{"-s", "examples/hello.sh"})
	if err != nil {
		t.Error("err not nil")
	}
	if cfg == nil {
		t.Error("cfg nil")
		return
	}
	if cfg.Steps == nil || len(cfg.Steps) != 1 {
		t.Error("steps")
	}
	script := cfg.Steps[0]
	if script == nil {
		t.Error("script nil")
	}
}

func TestMainParseNames(t *testing.T) {
	expectedNames := []string{"foo", "bar"}
	_, gotNames, _, _, err := parseArgs([]string{"-c", "true", "foo", "bar"})
	if err != nil {
		t.Error("err not nil")
	}
	if len(expectedNames) != len(gotNames) {
		t.Error("len(gotNames)")
	}
	for i, expectedName := range expectedNames {
		if expectedName != gotNames[i] {
			t.Error("expectedName")
		}
	}
}

func TestMainParseBadNames(t *testing.T) {
	_, gotNames, msg, status, _ := parseArgs([]string{"-c", "true", "user@foo", "bar"})
	if status == 0 {
		t.Error("status")
	}
	if len(gotNames) != 0 {
		t.Error("len(gotNames)")
	}
	if !strings.HasPrefix(msg, "error:") || !strings.HasSuffix(msg, "user@foo") {
		t.Error("msg")
	}
}

func TestMainParseExtraConfig(t *testing.T) {
	job, _, _, _, err := parseArgs([]string{"-F", "./ssh_config", "-c", "true"})
	if err != nil {
		t.Error("err not nil")
	}
	if len(job.SshArgs) != 2 ||
		job.SshArgs[0] != "-F" ||
		job.SshArgs[1] != "./ssh_config" {
		t.Error("job.SshArgs")
	}

	job, _, _, _, err = parseArgs([]string{"-c", "true"})
	if err != nil {
		t.Error("err not nil")
	}
	if len(job.SshArgs) != 0 {
		t.Error("job.SshArgs")
	}
}
