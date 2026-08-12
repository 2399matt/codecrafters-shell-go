package main

import (
	"fmt"
	"io"
	"os"
)

func (command *Command) prepCommand() (func(), func(), error) {
	command.out = terminalOut
	command.in = os.Stdin
	command.errOut = terminalErr
	parts := make([]string, 0, len(command.parts))
	var closeFunc func() = nil
	var errCloseFunc func() = nil
	var err error
	if len(command.parts) >= 1 && command.parts[len(command.parts)-1] == "&" {
		command.isJob = true
		command.parts = command.parts[:len(command.parts)-1]
	}
	for i := 0; i < len(command.parts); i++ {
		part := command.parts[i]
		if part == "2>>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			errCloseFunc, err = command.setupRedirect(path, os.O_APPEND, &command.errOut)
			if err != nil {
				return nil, nil, err
			}
			i++
			continue
		}
		if part == "2>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			errCloseFunc, err = command.setupRedirect(path, os.O_TRUNC, &command.errOut)
			if err != nil {
				return nil, nil, err
			}
			i++
			continue
		}
		if part == ">>" || part == "1>>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			closeFunc, err = command.setupRedirect(path, os.O_APPEND, &command.out)
			if err != nil {
				return nil, nil, err
			}
			i++
			continue
		}
		if part == ">" || part == "1>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
				return nil, nil, nil
			}
			path := command.parts[i+1]
			closeFunc, err = command.setupRedirect(path, os.O_TRUNC, &command.out)
			if err != nil {
				return nil, nil, err
			}
			i++
			continue
		}
		parts = append(parts, command.parts[i])
	}
	command.parts = parts
	return closeFunc, errCloseFunc, nil
}

func (command *Command) setupRedirect(path string, flag int, target *io.Writer) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: No such file or directory\n", command.keyword, path)
	}
	*target = file
	return func() { file.Close() }, nil
}
