package cli

import (
	"errors"
)

func GetCommands() map[string]Command {
	commands := []Command{
		DownloadCommand(),
		ConvertCommand(),
		UpdateCommand(),
	}

	commandsMap := make(map[string]Command)

	for _, command := range commands {
		commandsMap[command.Name] = command
	}

	return commandsMap
}

func ProcessCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("no command provided")
	}

	commands := GetCommands()

	command, isExist := commands[args[0]]

	if !isExist {
		return errors.New("Command not supported")
	}

	err := command.Run(args[1:])

	if err != nil {
		return err
	}

	return nil
}
