package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	//"github.com/chzyer/readline"
	"golang.org/x/term"
)

var keywords = []string{"echo", "exit", "type", "pwd", "cd", "complete", "jobs"}
var path string = os.Getenv("PATH")
var home string = os.Getenv("HOME")
var paths []string = strings.Split(path, ":")
var terminalOut io.Writer = TerminalWriter{os.Stdout}
var terminalErr io.Writer = TerminalWriter{os.Stderr}

func main() {
	buf := bufio.NewReader(os.Stdin)
	for {
		old, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		fmt.Print("$ ")
		buf.Reset(os.Stdin)
		var input string
		var awaitingMultiple bool = false
		for {
			b, err := buf.ReadByte()
			if err != nil {
				log.Fatalf("Error reading from console %v", err)
			}
			if b == '\t' {
				result := autocomplete(input, awaitingMultiple)
				awaitingMultiple = result.awaitingSuggestions
				switch result.action {
				case NoMatch:
					fmt.Printf("%c", 0x07)
				case SingleMatch:
					input = updateLastToken(input, result.suffix, result.addSpace)
					if result.addSpace {
						fmt.Printf("%s ", result.suffix)
					} else {
						fmt.Printf("%s", result.suffix)
					}
				case ShowSuggestions:
					fmt.Printf("\r\n")
					fmt.Printf("%s\r\n", strings.Join(result.suggestions, " "))
					fmt.Printf("$ %s", input)
				case PartialMatch:
					input = updateLastToken(input, result.suffix, result.addSpace)
					fmt.Printf("%s", result.suffix)
				default:
					continue
				}
				continue
			} else if b == 127 {
				if len(input) > 0 {
					awaitingMultiple = false
					input = input[:len(input)-1]
					fmt.Print("\b \b")
				}
				continue
			} else if b == '\r' || b == '\n' {
				fmt.Print("\r\n")
				break
			} else {
				awaitingMultiple = false
				input += string(b)
				fmt.Printf("%c", b)
				continue
			}
		}
		if err := term.Restore(int(os.Stdin.Fd()), old); err != nil {
			panic(err)
		}
		command, err := decodeCommand(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: command not found\r\n", input)
			continue
		}
		command.handleCommand()
		select {
		case job := <-cleanup:
			jobHandler.mu.Lock()
			if !job.reported {
				job.reported = true
				fmt.Fprintf(command.out, "[%d]+  %s                 %s\n", job.id, "Done", job.command)
				delete(jobHandler.jobs, job.id)
				jobHandler.runningId--
			}
			jobHandler.mu.Unlock()
		default:
		}
	}
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

func updateLastToken(currentInput string, added string, addSpace bool) string {
	parts := strings.Split(currentInput, " ")
	if len(parts) == 0 {
		parts = []string{""}
	}
	parts[len(parts)-1] += added
	joined := strings.Join(parts, " ")
	if addSpace && !strings.HasSuffix(joined, " ") {
		joined += " "
	}
	return joined
}
