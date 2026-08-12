package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type Completion struct {
	table map[string]CompletionSpec
}

// TODO will need to modify this for -W static lists.
type CompletionSpec struct {
	path string
}

func NewCompletion() *Completion {
	return &Completion{
		table: make(map[string]CompletionSpec),
	}
}

func (completion *Completion) extractSuggestions(rawInput, command, prevArg, path, prefix string) []string {
	cmd := exec.Command(path, command, prefix, prevArg)
	cmpLine := fmt.Sprintf("COMP_LINE=%s", rawInput)
	cmpPoint := fmt.Sprintf("COMP_POINT=%d", len(rawInput))
	cmd.Env = append(cmd.Env, cmpLine, cmpPoint)
	if out, err := cmd.Output(); err != nil {
		return nil
	} else {
		return strings.Fields(string(out))
	}
}

func (command *Command) handleComplete() {
	parts := command.parts
	flag := parts[0]
	switch flag {
	case "-p":
		if len(parts) >= 2 {
			if c, ok := completion.table[parts[1]]; ok {
				fmt.Fprintf(command.out, "complete -C '%s' %s\n", c.path, parts[1])
			} else {
				fmt.Fprintf(command.errOut, "%s: %s: no completion specification\n", command.keyword, parts[1])
			}
		}
	case "-C":
		if len(parts) >= 3 {
			key := parts[2]
			path := parts[1]
			completion.table[key] = CompletionSpec{path: path}
		}
	case "-r":
		if len(parts) >= 2 {
			key := parts[1]
			delete(completion.table, key)
		}
	}
}
