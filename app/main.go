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

func decodeCommand(command string) (*Command, error) {
	cmd := &Command{}
	arr := strings.Split(command, " ")
	if len(arr) == 0 {
		return nil, fmt.Errorf("No command")
	}
	if !slices.Contains(keywords, arr[0]) {
		for _, path := range paths {
			full := filepath.Join(path, arr[0])
			if checkPath(full) {
				cmd := exec.Command(full)
				if len(arr) > 1 {
					cmd.Args = append(cmd.Args, arr[1:]...)
				}
				if err := cmd.Run(); err != nil {
					fmt.Print("unable to execute\n")
					return nil, fmt.Errorf("Unexecutable")
				}
				return nil, nil
			}
		}
		return nil, fmt.Errorf("Unknown command")
	}
	cmd.keyword = arr[0]
	cmd.parts = arr[1:]
	return cmd, nil
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
			fmt.Printf("%s: not found\n", strInput)
			continue
		}
		handleCommand(command, paths)
	}
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
