package main

import (
	"os/exec"
	"strings"
)

type Completion struct {
	active bool
	path   string
	table  map[string]CompletionSpec
}

type CompletionSpec struct {
	path string
}

func NewCompletion() *Completion {
	return &Completion{
		active: false,
		table:  make(map[string]CompletionSpec),
	}
}

func (completion *Completion) extractSuggestions(path string) []string {
	cmd := exec.Command(path)
	if out, err := cmd.Output(); err != nil {
		return nil
	} else {
		return strings.Fields(string(out))
	}
}
