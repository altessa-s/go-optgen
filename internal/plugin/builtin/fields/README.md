# fields

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/fields"
```

Package `fields` implements functional-option field renderers.

## Plugins

| API | Purpose |
|-----|---------|
| `ManualPlugin` | Skips setters while keeping explicit defaults |
| `DefaultSetterPlugin` | Scalar and string setters |
| `SliceSetPlugin / DefaultAppenderPlugin` | Slice replacement or append |
| `BoolFlagPlugin / NetIPParsePlugin` | Boolean toggles and IP conversion |

## Usage

```go
err := fields.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
