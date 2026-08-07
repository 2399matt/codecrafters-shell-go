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
	"unicode"
)

var keywords = []string{"echo", "exit", "type", "pwd", "cd"}
var path string = os.Getenv("PATH")
var home string = os.Getenv("HOME")
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
		handleCommand(command)
	}
}

func decodeCommand(command string) (*Command, error) {
	cmd := &Command{}
	parts := make([]string, 0)
	inSingleQuote := false
	inDoubleQuote := false
	var curr strings.Builder
	for i := 0; i < len(command); i++ {
		ch := command[i]
		if ch == '\\' && i != len(command)-1 {
			i++
			ch = command[i]
			curr.WriteByte(ch)
		}
		if ch == '"' {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if unicode.IsSpace(rune(ch)) && !inSingleQuote && !inDoubleQuote {
			if curr.Len() > 0 {
				parts = append(parts, curr.String())
				curr.Reset()
			}
			continue
		}
		curr.WriteByte(ch)
	}
	if curr.Len() > 0 {
		parts = append(parts, curr.String())
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("No command")
	}
	if !slices.Contains(keywords, parts[0]) {
		if handleExec(parts) {
			cmd.keyword = "exec"
			return cmd, nil
		} else {
			return nil, fmt.Errorf("Unknown command")
		}
	}
	cmd.keyword = parts[0]
	cmd.parts = parts[1:]
	return cmd, nil
}

func handleCommand(command *Command) {
	switch command.keyword {
	case "type":
		if len(command.parts) == 0 {
			fmt.Printf("Usage: 'type keyword'\n")
			return
		}
		if slices.Contains(keywords, command.parts[0]) {
			fmt.Printf("%s is a shell builtin\n", command.parts[0])
		} else {
			if handleTypeSearch(command) {
				return
			} else {
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
			fmt.Printf("%s\n", wd)
		}
	case "cd":
		if handleDirChange(command.parts[0]) {
			return
		} else {
			fmt.Printf("%s: %s: No such file or directory\n", command.keyword, command.parts[0])
		}
	}
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

func handleDirChange(path string) bool {
	var err error
	if path == "~" {
		if err = os.Chdir(home); err != nil {
			return false
		}
		return true
	}
	if !filepath.IsAbs(path) {
		if path, err = filepath.Abs(path); err != nil {
			return false
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		if err := os.Chdir(path); err != nil {
			return false
		}
	}
	return true
}

func handleTypeSearch(command *Command) bool {
	for _, path := range paths {
		full := filepath.Join(path, command.parts[0])
		if checkPath(full) {
			fmt.Printf("%s is %s\n", command.parts[0], full)
			return true
		}
	}
	return false
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
