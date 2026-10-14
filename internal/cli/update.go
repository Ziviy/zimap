package cli

import (
	"fmt"
	"zimap/internal/setup"
)

func UpdateCommand() Command {
	return Command{
		Name:        "update",
		Description: "Command to update tools",
		Run: func(args []string) error {
			fmt.Println("Updating tools...")
			err := setup.UpdateTools()
			if err != nil {
				return err
			}
			return nil
		},
	}
}
