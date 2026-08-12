package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var completion = NewCompletion()

type Pipeline struct {
	commands []*Command
}
type Command struct {
	isJob   bool
	keyword string
	parts   []string
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
}

// Need carriage bytes if write stream is set to raw terminal
// This or a helper that checks on each call
type TerminalWriter struct {
	w io.Writer
}

func (t TerminalWriter) Write(p []byte) (n int, err error) {
	output := bytes.ReplaceAll(p, []byte("\n"), []byte("\r\n"))
	if _, err := t.w.Write([]byte(output)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (command *Command) handleCommand() {
	closer, errCloser, err := command.prepCommand()
	if command.isJob {
		command.setupJob()
		return
	}
	if err != nil {
		fmt.Fprintln(command.errOut, err.Error())
		return
	}
	if closer != nil {
		defer closer()
	}
	if errCloser != nil {
		defer errCloser()
	}
	switch command.keyword {
	case "type":
		if len(command.parts) == 0 {
			fmt.Fprintf(command.out, "Usage: 'type keyword'\n")
			return
		}
		if slices.Contains(keywords, command.parts[0]) {
			fmt.Fprintf(command.out, "%s is a shell builtin\n", command.parts[0])
		} else {
			if command.handleTypeSearch() {
				return
			} else {
				fmt.Fprintf(command.errOut, "%s not found\n", command.parts[0])
			}
		}
	case "echo":
		fmt.Fprintf(command.out, "%s\n", strings.Join(command.parts, " "))
	case "exit":
		os.Exit(0)
	case "pwd":
		if wd, err := os.Getwd(); err == nil {
			fmt.Fprintf(command.out, "%s\n", wd)
		}
	case "cd":
		if handleDirChange(command.parts[0]) {
			return
		} else {
			fmt.Fprintf(command.errOut, "%s: %s: No such file or directory\n", command.keyword, command.parts[0])
		}
	case "complete":
		command.handleComplete()
	case "jobs":
		handleJobs(command)
	default:
		if !slices.Contains(keywords, command.keyword) {
			if command.handleExec() {
				return
			} else {
				fmt.Fprintf(command.errOut, "%s: command not found\n", command.keyword)
			}
		}
	}
}

func (command *Command) handleExec() bool {
	for _, path := range paths {
		full := filepath.Join(path, command.keyword)
		if command.checkPath(full) {
			cmd := exec.Command(command.keyword)
			cmd.Args = append(cmd.Args, command.parts...)
			cmd.Stdout = command.out
			cmd.Stderr = command.errOut
			if err := cmd.Run(); err != nil {
				// wtf
				return true
			} else {
				return true
			}
		}
	}
	return false
}

func (command *Command) handleTypeSearch() bool {
	for _, path := range paths {
		full := filepath.Join(path, command.parts[0])
		if command.checkPath(full) {
			fmt.Fprintf(command.out, "%s is %s\n", command.parts[0], full)
			return true
		}
	}
	return false
}

func (command *Command) checkPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
		return true
	}
	return false
}
