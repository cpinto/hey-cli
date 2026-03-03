# Style

We aim to write code that is a pleasure to read, and we have a lot of opinions about how to do it well. Writing great code is an essential part of our programming culture, and we deliberately set a high bar for every code change anyone contributes. We care about how code reads, how code looks, and how code makes you feel when you read it.

We love discussing code. If you have questions about how to write something, or if you detect some smell you are not quite sure how to solve, please ask away to other programmers. A Pull Request is a great way to do this.

When writing new code, unless you are very familiar with our approach, try to find similar code elsewhere to look for inspiration.

## Code organization

The `./cmd` directory contains the entry points for the CLI and TUI interfaces.
Each file in that directory is a main package that can be build into a program in the `./bin` directory. 

The `./internal` directory contains the internal packages that implement the core functionality of the project, such as authentication, API client, configuration management, and so on. We organize code into packages based on functionality and domain concepts.

Individual CLI commands go into `./internal/cmd`. Each command or sub-command has its own file there.

Each command is a struct that holds its `*cobra.Command` and any flag fields. The constructor is named `newXyzCommand()`, creates the struct, builds the cobra.Command inline, defines flags, and returns the struct pointer:

```go
type composeCommand struct {
	cmd     *cobra.Command
	to      string
	subject string
	message string
}

func newComposeCommand() *composeCommand {
	composeCommand := &composeCommand{}
	composeCommand.cmd = &cobra.Command{
		Use:   "compose",
		Short: "Compose a new message",
		RunE:  composeCommand.run,
	}

	composeCommand.cmd.Flags().StringVar(&composeCommand.to, "to", "", "Recipient email address(es)")
	composeCommand.cmd.Flags().StringVar(&composeCommand.subject, "subject", "", "Message subject (required)")
	composeCommand.cmd.Flags().StringVarP(&composeCommand.message, "message", "m", "", "Message body")

	return composeCommand
}
```

The `run` method is a receiver on the command struct. It always starts with `requireAuth()`, handles `jsonOutput` early, and returns an `error`:

```go
func (c *composeCommand) run(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	if jsonOutput {
		data, err := apiClient.Get("/path.json")
		if err != nil {
			return err
		}
		return printRawJSON(data)
	}

	// Formatted output using newTable() or fmt.Println
	return nil
}
```

Parent commands that only group subcommands (like `todo`, `journal`, `timetrack`, `habit`) have no `RunE` — they only call `AddCommand`:

```go
func newTodoCommand() *todoCommand {
	todoCommand := &todoCommand{}
	todoCommand.cmd = &cobra.Command{
		Use:   "todo",
		Short: "Manage todos",
	}

	todoCommand.cmd.AddCommand(newTodoListCommand().cmd)
	todoCommand.cmd.AddCommand(newTodoAddCommand().cmd)
	todoCommand.cmd.AddCommand(newTodoCompleteCommand().cmd)

	return todoCommand
}
```

Commands are registered in the `Execute()` function in `root.go` via `rootCmd.AddCommand(newXyzCommand().cmd)`. Global state shared across commands (`cfg`, `apiClient`, `jsonOutput`, `baseURL`) lives as package-level variables in `root.go` and is initialized in `PersistentPreRunE`.

## Methods ordering

We order methods in the following order:

1. `public static` methods
1. `private static` methods
2. `public instance` methods
3. `private instance` methods

## Invocation order

We order methods vertically based on their invocation order. This helps us to understand the flow of the code.

```go
func (c *Client) PublicMethodA() {
    c.privateMethodA()
    c.privateMethodB()
}

func (c *Client) PublicMethodB() {
    c.privateMethodC()
}

func (c *Client) privateMethodA() {
    // ...
}

func (c *Client) privateMethodB() {
    // ...
}

func (c *Client) privateMethodC() {
    // ...
}
```
