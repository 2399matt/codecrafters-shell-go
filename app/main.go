package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var keywords = []string{"echo", "exit", "type"}
var path string = os.Getenv("PATH")
var paths []string = strings.Split(path, ":")

type Command struct {
	keyword string
	parts   []string
}

func main() {
	scan := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		input, err := scan.ReadBytes('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
		strInput := strings.TrimSpace(string(input[:len(input)-1]))
		command, err := decodeCommand(strInput)
		if err != nil {
			fmt.Printf("%s: command not found\n", strInput)
			continue
		}
		handleCommand(command, paths)
	}
}

func decodeCommand(command string) (*Command, error) {
	cmd := &Command{}
	arr := strings.Split(command, " ")
	if len(arr) == 0 {
		return nil, fmt.Errorf("No command")
	}
	if !slices.Contains(keywords, arr[0]) {
		if handleExec(arr) {
			cmd.keyword = "exec"
			return cmd, nil
		} else {
			return nil, fmt.Errorf("Unknown command")
		}
	}
	cmd.keyword = arr[0]
	cmd.parts = arr[1:]
	return cmd, nil
}

func handleExec(input []string) bool {
	for _, path := range paths {
		full := filepath.Join(path, input[0])
		if checkPath(full) {
			cmd := exec.Command(input[0])
			if len(input) > 1 {
				cmd.Args = append(cmd.Args, input[1:]...)
			}
			if out, err := cmd.Output(); err != nil {
				return false
			} else {
				fmt.Printf("%s", out)
				return true
			}
		}
	}
	return false
}

func handleCommand(command *Command, paths []string) {
	switch command.keyword {
	case "type":
		if len(command.parts) == 0 {
			fmt.Printf("Usage: 'type keyword'\n")
			return
		}
		if slices.Contains(keywords, command.parts[0]) {
			fmt.Printf("%s is a shell builtin\n", command.parts[0])
		} else {
			found := false
			for _, path := range paths {
				full := filepath.Join(path, command.parts[0])
				if checkPath(full) {
					fmt.Printf("%s is %s\n", command.parts[0], full)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("%s not found\n", command.parts[0])
			}
		}
	case "echo":
		fmt.Printf("%s\n", strings.Join(command.parts, " "))
	case "exit":
		os.Exit(0)
	case "exec":
		return
	case "pwd":
		if wd, err := os.Getwd(); err == nil {
			fmt.Printf("%s", wd)
		}
	}
}

func checkPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
		return true
	}
	return false
}
