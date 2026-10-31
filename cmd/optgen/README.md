# optgen

```go
import optgen "github.com/altessa-s/go-optgen"
```

Generate functional option functions from struct field tags.

## Overview

`optgen` generates type-safe functional option functions from struct field tags. It supports defaults, validation, value transformation, and custom
plugins.

## Installation

```bash
go install github.com/altessa-s/go-optgen/cmd/optgen@latest
```

## Usage

Add a go:generate directive to your options file:

```go
//go:generate go run github.com/altessa-s/go-optgen/cmd/optgen generate

type options struct {
    logger     *slog.Logger  `opt:"Logger"`
    timeout    time.Duration `opt:"Timeout" optgen:"default=30*time.Second"`
    labels     []string      `opt:"Labels" optval:"trimspaces,lower,dedup"`
    custom     string        `opt:"-"` // skipped - implement manually
}
```

Run generation:

```bash
go generate ./...
```

## Tag Reference

| Tag | Purpose | Example |
|-----|---------|---------|
| `opt:"Name"` | Generate WithName option | `opt:"Logger"` |
| `opt:"-"` | Skip field generation | `opt:"-"` |
| `optgen:"default=..."` | Default value | `optgen:"default=30*time.Second"` |
| `optgen:"append"` | Slice append semantics | `optgen:"append"` |
| `optgen:"notnil"` | Nil-check guard | `optgen:"notnil"` |
| `optgen:"manual"` | Skip auto-generation | `optgen:"manual"` |
| `optgen:"parseip"` | Accept IP strings, parse into `[]net.IP` | `optgen:"parseip"` |
| `optval:"..."` | Value modifiers | `optval:"trimspaces,lower"` |
| `optcheck:"..."` | Validation checks | `optcheck:"required,minlen=3"` |

## Commands

### generate

Generate functional option functions:

```bash
optgen generate [flags]

Flags:
  -t, --type string          Struct type name, or a comma-separated list (default "options")
  -o, --output string        Output file (default "options_gen.go")
  -d, --directory string     Directory to scan (default ".")
      --option-type string   Name of the Option type (default "Option")
      --option-error         Generate Option as func(*T) error
      --skip-option-type     Do not emit the Option type definition
      --skip-default-func    Do not emit the defaultOptions function
      --skip-new-func        Do not emit the newOptions function
      --default-func-name    Override the default<TypeName> function name
      --new-func-name        Override the new<TypeName> function name
      --option-prefix        Prefix for generated option names (WithPrefixXxx)
      --all-fields           Include untagged fields; still honor opt names and opt:"-"
      --disable-plugins      Comma-separated plugin names to disable
      --formatter string     Formatter command (default "gofmt -w")
      --no-format            Disable formatting
      --dry-run              Preview output without writing
  -v, --verbose              Enable verbose output
      --no-config            Ignore .optgen.yaml
      --func-prefix          Option function prefix (default "With")
      --invert-func-prefix   Prefix of inverted bool flag options (default "Without")
      --string-setter        String option signature: generic (string or *string) or plain
      --keep-empty-strings   Assign empty strings instead of skipping them
      --check-failure        Failed optcheck without an error result: panic (default) or skip
      --no-default-modifiers Do not apply default modifiers (trimspaces, notnil, dedup, positive)
      --allow-formatter      Add a formatter command name to the allowlist (repeatable; CLI only)
      --validate             Check optcheck rules on the final options in the constructor
      --with-isset           Track applied options in a <type>FieldSet field; generate isSet
      --strict               Treat warnings as errors; nothing is written on a warning
      --defaults-from        Defaults start value: tag (default), var[=name], func[=name], none
      --slice-setter         Slice option parameter: variadic (default) or slice
      --exclude              Skip fields matching name patterns (path.Match), comma-separated
      --source-file          Search the struct only in this file, e.g. "$GOFILE"
```

Several types in one `--type` list derive each name from the type: `checkOptions` generates `check_options_gen.go` with option type
`CheckOption` and functions `defaultCheckOptions`/`newCheckOptions`. `--output`, `--option-type`, `--default-func-name` and `--new-func-name`
cannot be combined with several types.

Field tags and generated code:

| Feature | Declaration | Generated |
|---|---|---|
| Mandatory field | `addr string optgen:"mandatory"` | `newOptions(addr string, opts ...Option)`, no `WithAddr` |
| Final validation | `--validate` with `optcheck` rules | `validate<Type>(o)` called by `newOptions`; unset fields and defaults are checked |
| Applied options | `--with-isset` and a field `set optionsFieldSet` | `optionsFieldName` constants and `o.isSet(optionsFieldName)` |
| Defaults from code | `--defaults-from var` and `var defaultOptionsValue = options{...}` | `defaultOptions` copies the variable, then applies tag defaults |
| Duration literal | `optgen:"default=1m30s"` on `time.Duration` | `90 * time.Second` |

### list-plugins

List available plugins and modifiers:

```bash
optgen list-plugins
```

## Configuration

Configuration lookup starts in the generation directory and searches upward for the nearest `.optgen.yaml`.
It searches through nested modules up to and including the first directory containing `.git` (a directory or a worktree file),
never reading configuration above that boundary. A nested configuration takes precedence over a repository configuration.
Outside a Git repository, lookup stops at the nearest `go.mod`, or at the filesystem root if no `go.mod` exists.
Paths in `packages:` are relative to the directory containing the selected configuration, including for nested modules.

Create `.optgen.yaml` in your project root:

```yaml
defaults:
  all-fields: true           # Process all struct fields
  skip-option-type: false    # Skip Option type generation
  option-error: false        # Make Option return error
  formatter: "gofmt -w"      # Code formatter

packages:
  "data/cache":              # exact relative path, no globs
    all-fields: false        # Package-specific override
```

## Value Modifiers

| Modifier | Description | Types |
|----------|-------------|-------|
| `trimspaces` | Trim whitespace | string |
| `lower` | Convert to lowercase | string |
| `upper` | Convert to uppercase | string |
| `dedup` | Remove duplicates; enabled by default, disable with `nodup` | Slices of recognized comparable types, including primitives, pointers, time.Time, and time.Duration |
| `positive` | Ensure positive value | numeric |
| `nonempty` | Skip assignment for an empty string; remove empty elements from string slices | string, []string |
| `trimprefix=P` / `trimsuffix=S` | Trim a prefix / suffix | string |

`trimspaces` is applied to every string field by default; add `notrim` to `optval` to disable it.

## Validation Checks

| Check | Description | Example |
|-------|-------------|---------|
| `required` | Require a non-nil pointer or a non-empty string, slice, or map | `optcheck:"required"` |
| `minlen=N` | Minimum length | `optcheck:"minlen=3"` |
| `maxlen=N` | Maximum length | `optcheck:"maxlen=100"` |
| `oneof=[...]` | Enum validation | `optcheck:"oneof=[debug,info,warn]"` |
| `nonzero` | Non-zero value | `optcheck:"nonzero"` |

## Examples

### Basic Options

```go
type options struct {
    timeout time.Duration `opt:"Timeout" optgen:"default=30*time.Second"`
    retries int          `opt:"Retries" optgen:"default=3"`
    debug   bool         `opt:"Debug"`
}
```

### Slice Options

```go
type options struct {
    hosts   []string `opt:"Hosts" optgen:"append"`
    labels  []string `opt:"Labels" optval:"dedup,nonempty"`
}
```

### Validated Options

```go
type options struct {
    level    string `opt:"Level" optcheck:"required,oneof=[debug,info,warn,error]"`
    workers  int    `opt:"Workers" optcheck:"nonzero" optval:"positive"`
    endpoint string `opt:"Endpoint" optcheck:"required,minlen=5"`
}
```

## See Also

- [Architecture](../../docs/architecture.md) - Layer boundaries and dependency rules