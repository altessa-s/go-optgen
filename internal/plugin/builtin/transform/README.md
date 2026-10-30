# transform

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/transform"
```

Package `transform` implements string-expression transformations.

## Plugins

| API | Purpose |
|-----|---------|
| `lower / upper` | Case conversion |
| `trimspaces` | Whitespace trimming |
| `trimprefix / trimsuffix` | Parameterized prefix and suffix removal |

## Usage

```go
err := transform.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
