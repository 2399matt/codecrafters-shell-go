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

func (completion *Completion) extractSuggestions(command, prevArg, path, prefix string) []string {
	cmd := exec.Command(path, command, prefix, prevArg)
	currentLineEnv := strings.Join([]string{command, prevArg, prefix}, " ")
	cmpLine := fmt.Sprintf("COMP_LINE=%s", currentLineEnv)
	// definitely gonna break if not ascii
	cmpPoint := fmt.Sprintf("COMP_POINT=%d", len(currentLineEnv))
	cmd.Env = append(cmd.Env, cmpLine, cmpPoint)
	if out, err := cmd.Output(); err != nil {
		return nil
	} else {
		return strings.Fields(string(out))
	}
}
