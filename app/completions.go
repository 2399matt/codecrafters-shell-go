package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type Completion struct {
	active bool
	path   string
	table  map[string]CompletionSpec
}

// TODO will need to modify this for -W static lists.
type CompletionSpec struct {
	path string
}

func NewCompletion() *Completion {
	return &Completion{
		active: false,
		table:  make(map[string]CompletionSpec),
	}
}

func (completion *Completion) extractSuggestions(command, prevArg, path, prefix string) []string {
	cmd := exec.Command(path, command, prefix, prevArg)
	cmpLine := fmt.Sprintf("COMP_LINE=%s", strings.Join([]string{command, prevArg, prefix}, " "))
	cmpPoint := fmt.Sprintf("COMP_POINT=%d", len(cmpLine))
	cmd.Env = append(cmd.Env, cmpLine, cmpPoint)
	if out, err := cmd.Output(); err != nil {
		return nil
	} else {
		return strings.Fields(string(out))
	}
}
