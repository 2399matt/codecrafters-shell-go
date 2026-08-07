package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

var keywords = []string{"echo", "exit"}

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
		case "echo":
			fmt.Printf("%s\n", strings.Join(command.parts, " "))
		case "exit":
			os.Exit(0)
		}
	}
}
