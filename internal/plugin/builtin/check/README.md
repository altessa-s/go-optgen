# check

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/check"
```

Package `check` implements value-validation plugins.

## Plugins

| API | Purpose |
|-----|---------|
| `RequiredCheck / NonZeroCheck` | Presence and nonzero validation |
| `MinLenCheck / MaxLenCheck` | Length bounds |
| `OneOfCheck` | Allowed string expressions |

## Usage

```go
err := check.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
