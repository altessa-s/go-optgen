# application

```go
import "github.com/altessa-s/go-optgen/internal/application"
```

Package `application` coordinates generation through consumer-owned ports.

## Key types

| API | Purpose |
|-----|---------|
| `Service` | Coordinates one generation request |
| `Request / Artifact` | Operation input and output |
| `Source / Preparer / Renderer / Output` | Replaceable dependencies |

## Usage

```go
service := application.New(source, preparer, renderer, output)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
