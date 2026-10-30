# commands

```go
import "github.com/altessa-s/go-optgen/commands"
```

Package `commands` implements the cobra-based CLI for the optgen code generator. Provides three entry points: `New` returns the full command tree, `Run`
is a convenience wrapper for `main()`, and individual subcommands are available via `NewGenerate`/`NewListPlugins`.

## Entry points

| Function         | Description                                                                                 |
|------------------|---------------------------------------------------------------------------------------------|
| `New`            | Returns a `*cobra.Command` tree with the `generate` and `list-plugins` subcommands (`--version` flag on the root) |
| `Run`            | Sets the given args, executes the root command, and returns any error                       |
| `NewGenerate`    | Returns the `generate` subcommand for embedding in custom CLI trees                         |
| `NewListPlugins` | Returns the `list-plugins` subcommand for embedding in custom CLI trees                     |

## Options

| Option        | Purpose                                                                                     |
|---------------|---------------------------------------------------------------------------------------------|
| `WithPlugins` | Registers additional plugins in every command execution, after the built-in plugins        |

## Custom plugins

Plugins are compiled into a program that wraps the CLI; there is no runtime plugin loading:

```go
// tools/optgen/main.go
func main() {
	if err := commands.Run(os.Args[1:], commands.WithPlugins(myplugins.Register)); err != nil {
		os.Exit(1)
	}
}
```

```go
//go:generate go run ./tools/optgen generate
```

## Configuration

Configuration is loaded from `.optgen.yaml` (see `config.File`) unless `--no-config` is set. Explicit CLI flags override corresponding config
values.

Each execution assembles its own registry: the built-in plugins, the configured `type-defaults`, the plugins of `WithPlugins`, then
`--disable-plugins`, which fails on an unknown name. Verbose
progress lists every field with its final option name, prefix included.
