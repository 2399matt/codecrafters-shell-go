package main

import (
	"fmt"
	"io"
)

type History struct {
	entries []string
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
}

func (h *History) popEntry() string {
	entry := h.entries[len(h.entries)-1]
	h.entries = h.entries[:len(h.entries)-1]
	return entry
}
