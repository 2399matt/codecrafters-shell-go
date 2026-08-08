package main

import (
	"bufio"
	"fmt"
	"io"
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
	out     io.Writer
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
			fmt.Fprintf(command.out, "%s: command not found\n", strInput)
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
		if (!inSingleQuote) && (ch == '\\' && i != len(command)-1) {
			if inDoubleQuote {
				temp := command[i+1]
				if temp != '"' && temp != '\\' {
					continue
				}
			}
			i++
			ch = command[i]
			curr.WriteByte(ch)
			continue
		}
		if !inSingleQuote && ch == '"' {
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

func prepCommand(command *Command) (func(), error) {
	command.out = os.Stdout
	parts := make([]string, len(command.parts))
	var closeFunc func()
	for i := 0; i < len(parts); i++ {
		part := command.parts[i]
		if part == ">" || part == "1>" {
			path := filepath.Join(path, command.parts[i+1])
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", command.keyword, command.parts[i+1])
				return nil, err
			}
			command.out = file
			closeFunc = func() { file.Close() }
			i++
			continue
		}
		parts = append(parts, command.parts[i])
	}
	command.parts = parts
	return closeFunc, nil
}

func handleCommand(command *Command) {
	closer, err := prepCommand(command)
	if err != nil {
		return
	}
	if closer != nil {
		defer closer()
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
			if handleTypeSearch(command) {
				return
			} else {
				fmt.Fprintf(os.Stderr, "%s not found\n", command.parts[0])
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
			fmt.Fprintf(command.out, "%s\n", wd)
		}
	case "cd":
		if handleDirChange(command.parts[0]) {
			return
		} else {
			fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", command.keyword, command.parts[0])
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
			fmt.Fprintf(command.out, "%s is %s\n", command.parts[0], full)
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
