# generator

```go
import "github.com/altessa-s/go-optgen/internal/generator"
```

Package `generator` renders prepared option declarations using injected plugins.

## Key types

| API | Purpose |
|-----|---------|
| `Generator` | Produces formatted Go source |
| `FieldSelector` | Field selection and pipeline contract |
| `model.GenerateInput` | Prepared rendering request |

## Usage

```go
gen := generator.New(registry, plugin.NewPipeline(registry))
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
