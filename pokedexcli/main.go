package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")

		if ok := scanner.Scan(); !ok {
			if err := scanner.Err(); err != nil {
				println("error reading user input: %w", err)
				return
			}

			return
		}

		input := strings.ToLower(scanner.Text())
		args := cleanInput(input)

		if len(args) > 0 {
			fmt.Printf("Your command was: %s\n", args[0])
		}
	}
}
