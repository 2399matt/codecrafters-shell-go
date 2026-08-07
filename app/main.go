package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	scan := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		input, err := scan.ReadBytes('\n')
		command := strings.TrimSpace(string(input[:len(input)-1]))
		if err != nil {
			log.Fatalf("%v", err)
		}
		switch command {
		case "exit":
			os.Exit(0)
		default:
			fmt.Printf("%s: command not found\n", command)
		}
	}
}
