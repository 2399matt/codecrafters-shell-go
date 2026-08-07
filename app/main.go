package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var keywords = []string{"echo", "exit", "type"}

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
		return nil, fmt.Errorf("Unknown command")
	}
	cmd.keyword = arr[0]
	cmd.parts = arr[1:]
	return cmd, nil
}

func main() {
	scan := bufio.NewReader(os.Stdin)
	path := os.Getenv("PATH")
	paths := strings.Split(path, ":")
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
		switch command.keyword {
		case "type":
			if len(command.parts) == 0 {
				fmt.Printf("Usage: 'type keyword'\n")
				continue
			}
			if slices.Contains(keywords, command.parts[0]) {
				fmt.Printf("%s is a shell builtin\n", command.parts[0])
			} else {
				var isExec string
				for i := range paths {
					isExec = walkDir(command.parts[0], paths[i])
					if isExec != "" {
						break
					}
				}
				if isExec == "" {
					fmt.Printf("%s: not found\n", command.parts[0])
				} else {
					fmt.Printf("%s is %s", command.parts[0], isExec)
				}
			}
		case "echo":
			fmt.Printf("%s\n", strings.Join(command.parts, " "))
		case "exit":
			os.Exit(0)
		}
	}
}

func walkDir(target, path string) string {
	res := ""
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.Contains(info.Name(), target) {
				info, err := info.Info()
				if err != nil {
					return err
				}
				// 3 bits to signify exec properties
				if info.Mode()&0111 != 0 {
					res = path
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Issue walking: %v", err)
	}
	return res
}
