package main

import (
	"fmt"
	"io"
)

type History struct {
	entries []string
	cursor  int
}

var history = &History{entries: make([]string, 0)}

func (h *History) listHistory(w io.Writer, numEntries int) {
	start := 0
	if numEntries != 0 {
		start = len(h.entries) - numEntries
	}
	for i := start; i < len(h.entries); i++ {
		fmt.Fprintf(w, "\t%d %s\n", i+1, h.entries[i])
	}
}

func (h *History) pushEntry(entry string) {
	h.entries = append(h.entries, entry)
	h.cursor++
}

func (h *History) stepBack() string {
	if h.cursor > 0 {
		h.cursor--
	}
	return h.entries[h.cursor]
}

func (h *History) stepForward() string {
	if h.cursor < len(h.entries)-1 {
		h.cursor++
	}
	return h.entries[h.cursor]
}
