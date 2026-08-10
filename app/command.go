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

type Command struct {
	keyword string
	parts   []string
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

func (command *Command) prepCommand() (func(), func(), error) {
	command.out = terminalOut
	command.errOut = terminalErr
	parts := make([]string, 0, len(command.parts))
	var closeFunc func() = nil
	var errCloseFunc func() = nil
	for i := 0; i < len(command.parts); i++ {
		part := command.parts[i]
		if part == "2>>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s: No such file or directory\n", command.keyword, command.parts[i+1])
			}
			command.errOut = file
			errCloseFunc = func() { file.Close() }
			i++
			continue
		}
		if part == "2>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s: No such file or directory\n", command.keyword, command.parts[i+1])
			}
			command.errOut = file
			errCloseFunc = func() { file.Close() }
			i++
			continue
		}
		if part == ">>" || part == "1>>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s: No such file or directory\n", command.keyword, command.parts[i+1])
			}
			command.out = file
			closeFunc = func() { file.Close() }
			i++
			continue
		}
		if part == ">" || part == "1>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s: No such file or directory\n", command.keyword, command.parts[i+1])
			}
			command.out = file
			closeFunc = func() { file.Close() }
			i++
			continue
		}
		parts = append(parts, command.parts[i])
	}
	command.parts = parts
	return closeFunc, errCloseFunc, nil
}

func (command *Command) handleCommand() {
	closer, errCloser, err := command.prepCommand()
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

func (command *Command) handleComplete() {
	parts := command.parts
	flag := parts[0]
	switch flag {
	case "-p":
		if len(parts) >= 2 {
			if c, ok := completion.table[parts[1]]; ok {
				fmt.Fprintf(command.out, "complete -C '%s' %s\n", c.path, parts[1])
			} else {
				fmt.Fprintf(command.errOut, "%s: %s: no completion specification\n", command.keyword, parts[1])
			}
		}
	case "-C":
		if len(parts) >= 3 {
			key := parts[2]
			path := parts[1]
			completion.table[key] = CompletionSpec{path: path}
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
				// tf
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
