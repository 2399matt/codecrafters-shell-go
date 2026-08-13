package main

import (
	"fmt"
	"strings"
	"unicode"
)

// need this to return a slice of commands.
// if | is encountered, new command needs to be started
func decodeCommand(command string) (*Command, error) {
	cmd := &Command{}
	parts := make([]string, 0)
	inSingleQuote := false
	inDoubleQuote := false
	var curr strings.Builder
	for i := 0; i < len(command); i++ {
		ch := command[i]
		// for inline-variables, super fragile
		if (!inSingleQuote) && ch == '$' {
			braces := false
			i++
			start := i
			if i < len(command) && command[i] == '{' {
				i++
				start = i
				for i < len(command) && command[i] != '}' {
					i++
				}
				braces = true
			} else {
				for i < len(command) && (unicode.IsLetter(rune(command[i])) || unicode.IsDigit(rune(command[i])) || command[i] == '_') {
					i++
				}
			}
			key := command[start:i]
			if val, ok := declarations[key]; !ok {
				//log.Fatalf("Unknown variable found in parsing: %s\n", key)
				continue
			} else {
				curr.WriteString(val)
				if !braces {
					i--
				}
				continue
			}
		}
		if (!inSingleQuote) && (ch == '\\' && i != len(command)-1) {
			if inDoubleQuote {
				temp := command[i+1]
				if temp != '"' && temp != '\\' {
					continue
				}
			}
			i++
			ch = command[i]
			curr.WriteByte(ch)
			continue
		}
		if !inSingleQuote && ch == '"' {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if unicode.IsSpace(rune(ch)) && !inSingleQuote && !inDoubleQuote {
			if curr.Len() > 0 {
				parts = append(parts, curr.String())
				curr.Reset()
			}
			continue
		}
		curr.WriteByte(ch)
	}
	if curr.Len() > 0 {
		parts = append(parts, curr.String())
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("No command")
	}
	cmd.keyword = parts[0]
	cmd.parts = parts[1:]
	return cmd, nil
}
