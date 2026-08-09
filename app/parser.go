package main

import (
	"fmt"
	"strings"
	"unicode"
)

func decodeCommand(command string) (*Command, error) {
	cmd := &Command{}
	parts := make([]string, 0)
	inSingleQuote := false
	inDoubleQuote := false
	var curr strings.Builder
	for i := 0; i < len(command); i++ {
		ch := command[i]
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
