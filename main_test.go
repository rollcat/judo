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
	job, _, _, _, err := parseArgs([]string{"-c", "true"})
	if job == nil {
		t.Error("job nil")
	}
	if err != nil {
		t.Error("err not nil")
	}
	if job.Command == nil {
		t.Error("command nil")
	}
	if job.Script != nil {
		t.Error("script not nil")
	}
}

func TestMainParseScriptNotExistent(t *testing.T) {
	job, _, msg, _, _ := parseArgs([]string{"-s", "examples/notfound.sh"})
	if msg == "" {
		t.Error("no message")
	}
	if job != nil {
		t.Error("job not nil")
	}
}

func TestMainParseScript(t *testing.T) {
	job, _, _, _, err := parseArgs([]string{"-s", "examples/hello.sh"})
	if err != nil {
		t.Error("err not nil")
	}
	if job == nil {
		t.Error("job nil")
		return
	}
	if job.Script == nil {
		t.Error("script nil")
	}
	if job.Command != nil {
		t.Error("command not nil")
	}
	if job.Script.IsDirMode() {
		t.Error("unexpected dirmode")
	}
}

func TestMainParseScriptDirMode(t *testing.T) {
	job, _, _, _, err := parseArgs([]string{"-s", "examples/bootstrap"})
	if err != nil {
		t.Error("err not nil")
	}
	if job == nil {
		t.Error("job nil")
		return
	}
	if job.Script == nil {
		t.Error("script nil")
	}
	if job.Command != nil {
		t.Error("command not nil")
	}
	if !job.Script.IsDirMode() {
		t.Error("expected dirmode")
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
