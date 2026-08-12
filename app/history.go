package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type History struct {
	writeFile string
	readFile  string
	entries   []string
	cursor    int
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

func (h *History) writeToFile(path string) {
	h.writeFile = path
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	for _, entry := range h.entries {
		fmt.Fprintf(file, "%s\n", entry)
	}
}

func (h *History) readFromFile(path string) {
	h.readFile = path
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		line := scan.Text()
		h.entries = append(h.entries, line)
	}
	if err := scan.Err(); err != nil {
		return
	}
}
