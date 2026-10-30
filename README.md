# go-optgen

```sh
go install github.com/altessa-s/go-optgen/cmd/optgen@latest
```

`optgen` is a code generator for the functional option pattern. It scans Go structs for fields annotated with `opt`/`optgen`/`optval`/`optcheck` tags
and generates type-safe `WithXxx` option functions, default constructors, and validation code with an extensible plugin system.

## Tag format

| Tag        | Purpose                                                                          |
|------------|----------------------------------------------------------------------------------|
| `opt`      | Option name for the `WithXxx` function; use `-` to skip generation. Omit it to derive the name from the field (see [Option naming](#option-naming)) |
| `optgen`   | Generator directives: `default=...`, `append`, `notnil`, `manual`, `parseip`, `err=...` |
| `optval`   | Value modifiers: `trimspaces`, `lower`, `upper`, `trimprefix=...`, `trimsuffix=...`, `dedup`, `nonempty`, `positive`; `invert` / `param` for bool fields. `trimspaces` applies to string fields by default; `notrim` disables it |
| `optcheck` | Validation rules: `required`, `nonzero`, `minlen=N`, `maxlen=N`, `oneof=[a,b,c]`|

## Option naming

For fields selected by `--all-fields`, `all-fields: true`, or an `optgen`/`optval`/`optcheck` tag, an omitted `opt` tag derives the option name from the
field name, with every word that is a known initialism upper-cased — Go style requires an initialism to keep a consistent case throughout an identifier:

| Field           | Generated                |
|-----------------|--------------------------|
| `ttl`           | `WithTTL`                |
| `httpClient`    | `WithHTTPClient`         |
| `entityId`      | `WithEntityIDExtractor`* |
| `maxIdleTime`   | `WithMaxIdleTime`        |
| `grpcOptions`   | `WithGrpcOptions`        |

<sub>* for a field named `entityIdExtractor`.</sub>

The table is golint's `commonInitialisms` plus `DB`. `gRPC` is deliberately absent: the repo spells it `Grpc` throughout (`grpcconfig.Config`,
`DefaultGrpc`), and a lone `WithGRPCOptions` would be less consistent, not more. To override a spelling the table gets wrong, name the option explicitly
— the `opt` tag always wins:

```go
type options struct {
    ttl time.Duration                // -> WithTTL
    uid string        `opt:"Uid"`    // -> WithUid
}
```

## CLI commands

| Command        | Description                                                               |
|----------------|---------------------------------------------------------------------------|
| `generate`     | Parse the struct and generate option functions into `*_gen.go`            |
| `list-plugins` | List registered field plugins, type-default providers, transform plugins, modifier plugins, and check plugins with their keys and aliases                     |

The root command prints version information with `--version`.

## Configuration

Settings are loaded from `.optgen.yaml`, searched from `--directory` (default `.`) up to the project root. Explicit CLI flags override corresponding
config values. Per-package overrides use the `packages` map keyed by relative path; package-specific `disable-plugins` overrides global defaults.

## Library API

```go
import optgen "github.com/altessa-s/go-optgen"
```

The root package implements the CLI; `cmd/optgen` is a thin `main` around `optgen.Run`.

### Entry points

| Function         | Description                                                                                 |
|------------------|---------------------------------------------------------------------------------------------|
| `New`            | Returns a `*cobra.Command` tree with the `generate` and `list-plugins` subcommands (`--version` flag on the root) |
| `Run`            | Sets the given args, executes the root command, and returns any error                       |
| `NewGenerate`    | Returns the `generate` subcommand for embedding in custom CLI trees                         |
| `NewListPlugins` | Returns the `list-plugins` subcommand for embedding in custom CLI trees                     |

### Options

| Option        | Purpose                                                                                     |
|---------------|---------------------------------------------------------------------------------------------|
| `WithPlugins` | Registers additional plugins in every command execution, after the built-in plugins        |

### Custom plugins

Plugins are compiled into a program that wraps the CLI; there is no runtime plugin loading (complete example:
[examples/customplugin](./examples/customplugin)):

```go
// tools/optgen/main.go
func main() {
	if err := optgen.Run(os.Args[1:], optgen.WithPlugins(myplugins.Register)); err != nil {
		os.Exit(1)
	}
}
```

```go
//go:generate go run ./tools/optgen generate
```

### Configuration

Configuration is loaded from `.optgen.yaml` (see `config.File`) unless `--no-config` is set. Explicit CLI flags override corresponding config
values.

Each execution assembles its own registry: the built-in plugins, the configured `type-defaults`, the plugins of `WithPlugins`, then
`--disable-plugins`, which fails on an unknown name. Verbose
progress lists every field with its final option name, prefix included.

## Subpackages

| Package                      | Description                                                           |
|------------------------------|-----------------------------------------------------------------------|
| [cmd/optgen](./cmd/optgen)   | `optgen` binary: a `main` around `optgen.Run`; CLI flag reference     |
| [config](./config)           | Loads and merges `.optgen.yaml` configuration with per-package overrides|
| [model](./model)             | Public data structures (`OptField`, `GenericInfo`) shared across layers|
| [plugin](./plugin)           | Public extension API: `FieldPlugin`, `TransformPlugin`, `Registry`    |
| `internal/application`       | Generation use case coordinating source, preparation, rendering, output|
| `internal/parser`            | Syntax extraction of struct fields, tags, and imports                 |
| `internal/semantic`          | Validation of checks and application of catalog default policies      |
| `internal/generator`         | Renders prepared option models through plugin contracts               |
| `internal/naming`            | Derives option names from field names, with initialism handling       |
| `internal/source`            | Filesystem AST source adapter                                         |
| `internal/output`            | Atomic output publication and allowlisted code formatting             |
| [examples](./examples)       | Runnable examples: strict validation, generics, several types, custom plugins |

## Architecture

SOLID and Clean Architecture are mandatory. See [architecture](docs/architecture.md) for the layer boundaries, plugin contracts, and verification rules.
