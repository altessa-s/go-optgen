# source

```go
import "github.com/altessa-s/go-optgen/internal/source"
```

Package `source` reads Go packages and extracts declarations. It loads the package with `go/packages` so field types are type-checked
(compile errors such as not-yet-generated options are tolerated) and falls back to syntax-only parsing outside a module.

## Key types

| API | Purpose |
|-----|---------|
| `Reader` | Filesystem implementation of the source port |
| `Parse` | Skips tests and generated source files (by the `Code generated` header) |

## Usage

```go
packages, err := (source.Reader{}).Parse(ctx, dir, "options", true)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
