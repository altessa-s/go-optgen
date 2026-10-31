# semantic

```go
import "github.com/altessa-s/go-optgen/internal/semantic"
```

Package `semantic` validates declarations and applies explicit plugin policies: the option name prefix, check applicability, type defaults and
default normalization, and the default modifiers derived from plugin metadata. `PrepareInput` also applies the settings policies of a rendering
request (`defaults-from=none`, `with-isset`).

## Options

| Option            | Default | Purpose                                                      |
|-------------------|---------|--------------------------------------------------------------|
| `WithPrefix`      | `""`    | Prepended to every option name before validation             |
| `WithWarningSink` | `nil`   | Receives diagnostics, such as a check of a disabled plugin   |
| `WithExclude` | none | Drops fields whose name matches a `path.Match` pattern |
| `WithDefaultModifiers` | `true` | Applies the modifiers plugins declare for a type (trimspaces, notnil, ...) |

## Key types

| API        | Purpose                                                                         |
|------------|---------------------------------------------------------------------------------|
| `Catalog`  | Required policy queries: plugin descriptors, type defaults and their normalization, check plugins |
| `Preparer` | Implements the application preparation port; returns validated, enriched copies |

## Usage

```go
prepared, err := semantic.New(registry, semantic.WithPrefix("Validation")).Prepare(fields)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
