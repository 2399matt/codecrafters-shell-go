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
	"unicode"
)

// TODO for pipelines, we need to have a slice of commands. Parser can return this slice.
// pipes will have to be setup for the slice of commands out(n) -> in(n+1), then handleCommand can be called with goroutines per command, don't think it will work if sync.
// jobs will also be fragile, & will need to register an entire pipe command, not just parts (no ideas atm)
var completion = NewCompletion()

type Command struct {
	isJob   bool
	keyword string
	parts   []string
	// in for pipeline addition
	in     io.Reader
	out    io.Writer
	errOut io.Writer
}

// Need carriage byte if write stream is set to raw terminal
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
		history.writeToFile(HistoryFile)
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
	case "history":
		handleHistory(command)
	case "declare":
		command.handleDeclare()
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

func (command *Command) handleDeclare() {
	if len(command.parts) == 0 {
		return
	}
	flag := command.parts[0]
	switch flag {
	case "-p":
		name := command.parts[1]
		if val, ok := declarations[name]; !ok {
			fmt.Fprintf(command.errOut, "%s: %s: not found\n", command.keyword, name)
		} else {
			fmt.Fprintf(command.out, "%s -- %s=\"%s\"\n", command.keyword, name, val)
		}
	default:
		declaration := strings.Split(command.parts[0], "=")
		name := declaration[0]
		if !validateVarName(name) {
			fmt.Fprintf(command.errOut, "%s: `%s': not a valid identifier\n", command.keyword, command.parts[0])
			return
		}
		val := declaration[1]
		declarations[name] = val
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
			cmd.Stdin = command.in
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

func validateVarName(varname string) bool {
	if unicode.IsDigit(rune(varname[0])) {
		return false
	}
	for i := 1; i < len(varname); i++ {
		ch := rune(varname[i])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}
