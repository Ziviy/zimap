package main

import (
	"fmt"
	"zimap/internal/cli"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Println("Command not stated")
		os.Exit(1)
	}

	err := cli.ProcessCommand(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing command\n%v\n", err)
		os.Exit(1)
	}
}
