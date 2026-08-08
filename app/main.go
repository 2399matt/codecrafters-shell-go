package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	term "golang.org/x/term"
)

var keywords = []string{"echo", "exit", "type", "pwd", "cd"}
var path string = os.Getenv("PATH")
var home string = os.Getenv("HOME")
var paths []string = strings.Split(path, ":")

type Command struct {
	keyword string
	parts   []string
	out     io.Writer
	errOut  io.Writer
}

func main() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	buf := make([]byte, 1)
	input := make([]byte, 0, 255)
	fmt.Print("$ ")
	for {
		_, err := os.Stdin.Read(buf)
		if err != nil {
			panic(err)
		}
		switch buf[0] {
		case '\t':
			// autocomplete
			prefix := string(input)
			for _, keyword := range keywords {
				if strings.HasPrefix(keyword, prefix) {
					suffix := keyword[len(prefix):]
					suffix += " "
					fmt.Print(suffix)
					input = append(input, suffix...)
					break
				}
			}
		case '\r':
			fmt.Print("\r\n")
			cmd := strings.TrimSpace(string(input))
			command, err := decodeCommand(cmd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd)
				fmt.Print("\b\b\b\b")
				continue
			}
			handleCommand(command)
			for range len(input) {
				fmt.Print("\b")
			}
			input = input[:0]
			fmt.Print("$ ")
			// enter (done) - process input string
		case 127:
			//backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
		default:
			input = append(input, buf[0])
			fmt.Printf("%c", buf[0])
		}
	}
	// scan := bufio.NewReader(os.Stdin)
	// for {
	// 	fmt.Print("$ ")
	// 	input, err := scan.ReadBytes('\n')
	// 	if err != nil {
	// 		log.Fatalf("Error reading input: %v", err)
	// 	}
	// 	strInput := strings.TrimSpace(string(input[:len(input)-1]))
	// 	command, err := decodeCommand(strInput)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "%s: command not found\n", strInput)
	// 		continue
	// 	}
	// 	handleCommand(command)
	// }
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
	cmd.keyword = parts[0]
	cmd.parts = parts[1:]
	return cmd, nil
}

func prepCommand(command *Command) (func(), func(), error) {
	command.out = os.Stdout
	command.errOut = os.Stderr
	parts := make([]string, 0, len(command.parts))
	var closeFunc func() = nil
	var errCloseFunc func() = nil
	for i := 0; i < len(command.parts); i++ {
		part := command.parts[i]
		if part == "2>>" {
			if i >= len(command.parts)-1 {
				fmt.Fprintln(command.errOut, "No specified file")
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

func handleCommand(command *Command) {
	closer, errCloser, err := prepCommand(command)
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
			if handleTypeSearch(command) {
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
	default:
		if !slices.Contains(keywords, command.keyword) {
			if handleExec(command) {
				return
			} else {
				fmt.Fprintf(command.errOut, "%s: command not found\n", command.keyword)
			}
		}
	}
}

func handleExec(command *Command) bool {
	for _, path := range paths {
		full := filepath.Join(path, command.keyword)
		if checkPath(full) {
			cmd := exec.Command(command.keyword)
			cmd.Args = append(cmd.Args, command.parts...)
			cmd.Stdout = command.out
			cmd.Stderr = command.errOut
			if err := cmd.Run(); err != nil {
				return true
			} else {
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
