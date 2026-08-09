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
	"sort"
	"strings"
	"unicode"

	//"github.com/chzyer/readline"
	"golang.org/x/term"
)

var keywords = []string{"echo", "exit", "type", "pwd", "cd"}
var path string = os.Getenv("PATH")
var home string = os.Getenv("HOME")
var paths []string = strings.Split(path, ":")
var terminalOut io.Writer = TerminalWriter{os.Stdout}
var terminalErr io.Writer = TerminalWriter{os.Stderr}

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
	output := string(p)
	output = strings.ReplaceAll(output, "\n", "\r\n")
	return t.w.Write([]byte(output))
}

func main() {
	old, err := term.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), old)
	buf := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		buf.Reset(os.Stdin)
		var input string
		var isFirstWord bool = true
		var multipleFiles bool = false
		fileResults := make([]string, 0)
		for {
			b, err := buf.ReadByte()
			if err != nil {
				log.Fatalf("Error reading from console %v", err)
			}
			if b == '\t' {
				if keyword, ok := searchKeyword(input); ok {
					suffix := keyword[len(input):]
					input += suffix
					input += " "
					fmt.Printf("%s ", suffix)
					isFirstWord = false
					continue
				}
				toMatch := input
				if !isFirstWord {
					idx := strings.LastIndex(input, " ")
					if idx != -1 && idx < len(input) {
						toMatch = input[idx+1:]
					}
				}
				fileResults = searchForFileMatch(toMatch)
				matches := len(fileResults)
				if matches == 0 {
					multipleFiles = false
					fmt.Printf("%c", 0x07)
				} else if matches == 1 {
					multipleFiles = false
					suffix := fileResults[0][len(input):]
					input += suffix
					input += " "
					fmt.Printf("%s ", suffix)
				} else {
					lcp := longestCommonPrefix(fileResults)

					if len(lcp) > len(input) {
						// Multiple matches, but we can still autocomplete.
						suffix := lcp[len(input):]
						input += suffix
						fmt.Print(suffix)

						// We aren't waiting for another tab yet.
						multipleFiles = false
					} else if multipleFiles {
						// LCP == input, so the second Tab lists the possibilities.
						fmt.Printf("\r\n")
						fmt.Printf("%s\r\n", strings.Join(fileResults, " "))
						fmt.Printf("$ %s", input)
						multipleFiles = false
					} else {
						// LCP == input, first Tab: bell.
						fmt.Printf("%c", 0x07)
						multipleFiles = true
					}
				}
				continue
			} else if b == 127 {
				if len(input) > 0 {
					// assuming 1-byte?
					input = input[:len(input)-1]
					fmt.Print("\b \b")
				}
				continue
			} else if b == '\r' || b == '\n' {
				fmt.Print("\r\n")
				break
			} else {
				input += string(b)
				fmt.Printf("%c", b)
				continue
			}
		}
		//input = strings.TrimSpace(input)
		command, err := decodeCommand(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: command not found\r\n", input)
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
	cmd.keyword = parts[0]
	cmd.parts = parts[1:]
	return cmd, nil
}

func prepCommand(command *Command) (func(), func(), error) {
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

func longestCommonPrefix(files []string) string {
	copied := make([]string, len(files))
	copy(copied, files)
	sort.Slice(copied, func(i, j int) bool {
		return len(copied[i]) < len(copied[j])
	})
	shortest := copied[0]
	res := ""
	for _, str := range files {
		if len(str) < len(shortest) {
			shortest = str
		}
	}
	for i := 0; i < len(shortest); i++ {
		ch := shortest[i]
		for _, str := range files {
			if i < len(str) && str[i] != ch {
				return res
			}
		}
		res += string(ch)
	}
	return res
}

func searchKeyword(prefix string) (string, bool) {
	for _, kw := range keywords {
		if strings.HasPrefix(kw, prefix) {
			return kw, true
		}
	}
	return "", false
}

func searchForFileMatch(prefix string) []string {
	results := make([]string, 0)
	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasPrefix(file.Name(), prefix) {
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Mode().Perm()&0o111 != 0 {
				results = append(results, file.Name())
			}
		}
	}
	slices.Sort(results)
	return results
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
