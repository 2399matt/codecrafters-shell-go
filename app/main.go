package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	scan := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		input, err := scan.ReadBytes('\n')
		command := input[:len(input)-1]
		if err != nil {
			log.Fatalf("%v", err)
		}
		switch command {
		default:
			fmt.Printf("%s: command not found\n", command)
		}
	}
}
