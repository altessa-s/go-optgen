# parser

```go
import "github.com/altessa-s/go-optgen/internal/parser"
```

Package `parser` extracts opt-tagged declarations without applying plugin policies.

## Functions

| API | Purpose |
|-----|---------|
| `FindOptFields` | Extracts fields and imports; an optional `TypeResolver` supplies exact `TypeFacts` |
| `ParseOptGen / ParseExprList` | Parse tag declarations |
| `FindNormalizationMethod` | Finds a normalization method |

## Usage

```go
result, err := parser.FindOptFields(pkg, "options", true, nil)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
