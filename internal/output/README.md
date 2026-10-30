# output

```go
import "github.com/altessa-s/go-optgen/internal/output"
```

Package `output` formats generated source and publishes successful output atomically.

## Key types

| API | Purpose |
|-----|---------|
| `Writer` | Filesystem implementation of the output port |
| `New` | Binds process output streams and diagnostics; `WithAllowedFormatters` extends the formatter allowlist |
| `Write` | Formats a temporary file before rename |

## Usage

```go
writer := output.New(stdout, stderr, diagnostics)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
