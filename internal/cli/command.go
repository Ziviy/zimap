package cli

type Command struct {
	Name        string
	Description string
	Run         func(args []string) error
}
