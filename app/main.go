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
		input, _, err := scan.ReadLine()
		if err != nil {
			log.Fatalf("%v", err)
		}
		switch input {
		default:
			fmt.Printf("%s: command not found\n", string(input))
		}
	}
}
