package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type cliCommand struct {
	name, description string
	callback          func(*cliConfig) error
}

type cliConfig struct {
	commands map[string]cliCommand
}

func main() {
	config := cliConfig{commands: map[string]cliCommand{
		"exit": cliCommand{
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": cliCommand{
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
	}}

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

		if len(args) == 0 {
			continue
		}

		command, ok := config.commands[args[0]]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		err := command.callback(&config)
		if err != nil {
			fmt.Printf("Error executing %q command: %s\n", command.name, err.Error())
		}
	}
}

func commandExit(_ *cliConfig) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)

	return nil
}

func commandHelp(c *cliConfig) error {
	if len(c.commands) == 0 {
		return fmt.Errorf("No commands registered")
	}

	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()

	for _, command := range c.commands {
		fmt.Printf("%s: %s\n", command.name, command.description)
	}

	return nil
}
